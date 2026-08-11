// Copyright 2026 opslog Author. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//      http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package input

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"

	ingestpb "github.com/go-the-way/opslog/pkg/transport/grpctx/ingest"
	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/server/contract"
	disp "github.com/go-the-way/opslog/server/internal/dispatcher"
)

type GRPC struct {
	name       string
	listen     string
	decoder    contract.Decoder
	dispatcher contract.Dispatcher
	srv        *grpc.Server
	ln         net.Listener
}

func NewGRPC(name string, cfg map[string]any, decoder contract.Decoder) (contract.Input, error) {
	if decoder == nil {
		return nil, fmt.Errorf("grpc input: decoder required")
	}
	if name == "" {
		name = "grpc"
	}
	return &GRPC{
		name:    name,
		listen:  cfgutil.String(cfg, "listen", ":8900"),
		decoder: decoder,
	}, nil
}

func (g *GRPC) Name() string { return g.name }
func (g *GRPC) Type() string { return "grpc" }

func (g *GRPC) Start(ctx context.Context, dispatcher contract.Dispatcher) error {
	ln, err := net.Listen("tcp", g.listen)
	if err != nil {
		return err
	}
	g.ln = ln
	g.dispatcher = dispatcher
	g.srv = grpc.NewServer()
	ingestpb.RegisterIngestServer(g.srv, &ingestServer{parent: g})
	go func() { _ = g.srv.Serve(ln) }()
	go func() {
		<-ctx.Done()
		_ = g.Stop(context.Background())
	}()
	return nil
}

func (g *GRPC) Stop(context.Context) error {
	if g.srv != nil {
		g.srv.GracefulStop()
	}
	return nil
}

type ingestServer struct {
	ingestpb.UnimplementedIngestServer
	parent *GRPC
}

func grpcRemote(ctx context.Context) string {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String()
	}
	return "grpc"
}

func (s *ingestServer) Send(ctx context.Context, req *ingestpb.SendRequest) (*ingestpb.SendReply, error) {
	remote := grpcRemote(ctx)
	sig, err := s.parent.decoder.Decode(remote, req.GetPayload())
	if err != nil {
		return nil, err
	}
	if err := s.parent.dispatcher.Dispatch(disp.WithMeta(ctx, s.parent.name, remote), sig); err != nil {
		return nil, err
	}
	return &ingestpb.SendReply{Accepted: 1}, nil
}

func (s *ingestServer) SendStream(stream ingestpb.Ingest_SendStreamServer) error {
	var n int32
	ctx := stream.Context()
	remote := grpcRemote(ctx)
	for {
		req, err := stream.Recv()
		if err != nil {
			return stream.SendAndClose(&ingestpb.SendReply{Accepted: n})
		}
		sig, err := s.parent.decoder.Decode(remote, req.GetPayload())
		if err != nil {
			continue
		}
		if err := s.parent.dispatcher.Dispatch(disp.WithMeta(ctx, s.parent.name, remote), sig); err == nil {
			n++
		}
	}
}

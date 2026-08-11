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

package transport

import (
	pkgtransport "github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/pkg/transport/grpctx"
	"github.com/go-the-way/opslog/pkg/transport/httpx"
	"github.com/go-the-way/opslog/pkg/transport/tcp"
	"github.com/go-the-way/opslog/pkg/transport/udp"
	"github.com/go-the-way/opslog/pkg/transport/ws"
)

func NewUDPTransport(name, addr string) (pkgtransport.Transport, error) {
	return udp.New(name, map[string]any{"addr": addr})
}

func NewTCPTransport(name, addr string) (pkgtransport.Transport, error) {
	return tcp.New(name, map[string]any{"addr": addr})
}

// NewHTTPTransport builds an HTTP ingest transport.
// Optional token is sent as Authorization: Bearer <token> when non-empty.
func NewHTTPTransport(name, url string, token ...string) (pkgtransport.Transport, error) {
	cfg := map[string]any{"url": url}
	if len(token) > 0 && token[0] != "" {
		cfg["token"] = token[0]
	}
	return httpx.New(name, cfg)
}

// NewWebSocketTransport builds a WebSocket ingest transport.
// Optional token is sent as Authorization: Bearer <token> when non-empty.
func NewWebSocketTransport(name, url string, token ...string) (pkgtransport.Transport, error) {
	cfg := map[string]any{"url": url}
	if len(token) > 0 && token[0] != "" {
		cfg["token"] = token[0]
	}
	return ws.New(name, cfg)
}

// NewGRPCTransport builds a gRPC ingest transport.
// Optional token is sent as authorization metadata Bearer <token> when non-empty.
func NewGRPCTransport(name, addr string, token ...string) (pkgtransport.Transport, error) {
	cfg := map[string]any{"addr": addr}
	if len(token) > 0 && token[0] != "" {
		cfg["token"] = token[0]
	}
	return grpctx.New(name, cfg)
}

func NewTransport(typ pkgtransport.Type, name string, cfg map[string]any) (pkgtransport.Transport, error) {
	switch typ {
	case pkgtransport.TypeUDP:
		return udp.New(name, cfg)
	case pkgtransport.TypeTCP:
		return tcp.New(name, cfg)
	case pkgtransport.TypeHTTP:
		return httpx.New(name, cfg)
	case pkgtransport.TypeWebSocket:
		return ws.New(name, cfg)
	case pkgtransport.TypeGRPC:
		return grpctx.New(name, cfg)
	default:
		return udp.New(name, cfg)
	}
}

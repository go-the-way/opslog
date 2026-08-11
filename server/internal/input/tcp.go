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
	"sync"
	"time"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/framer"
	"github.com/go-the-way/opslog/server/contract"
	disp "github.com/go-the-way/opslog/server/internal/dispatcher"
)

type TCP struct {
	name    string
	listen  string
	decoder contract.Decoder
	ln      net.Listener
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func NewTCP(name string, cfg map[string]any, decoder contract.Decoder) (contract.Input, error) {
	listen := cfgutil.String(cfg, "listen", ":8141")
	if name == "" {
		name = "tcp"
	}
	if decoder == nil {
		return nil, fmt.Errorf("tcp input: decoder required")
	}
	return &TCP{name: name, listen: listen, decoder: decoder}, nil
}

func (t *TCP) Name() string { return t.name }
func (t *TCP) Type() string { return "tcp" }

func (t *TCP) Start(ctx context.Context, dispatcher contract.Dispatcher) error {
	ln, err := net.Listen("tcp", t.listen)
	if err != nil {
		return err
	}
	t.ln = ln
	cctx, cancel := context.WithCancel(ctx)
	t.cancel = cancel
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				if cctx.Err() != nil {
					return
				}
				continue
			}
			t.wg.Add(1)
			go t.handle(cctx, conn, dispatcher)
		}
	}()
	return nil
}

func (t *TCP) handle(ctx context.Context, conn net.Conn, dispatcher contract.Dispatcher) {
	defer t.wg.Done()
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		payload, err := framer.ReadLengthPrefixed(conn)
		if err != nil {
			return
		}
		sig, err := t.decoder.Decode(remote, payload)
		if err != nil {
			continue
		}
		_ = dispatcher.Dispatch(disp.WithMeta(ctx, t.name, remote), sig)
	}
}

func (t *TCP) Stop(context.Context) error {
	if t.cancel != nil {
		t.cancel()
	}
	if t.ln != nil {
		_ = t.ln.Close()
	}
	t.wg.Wait()
	return nil
}

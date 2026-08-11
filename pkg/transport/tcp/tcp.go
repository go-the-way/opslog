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

package tcp

import (
	"context"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/framer"
	"github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/pkg/transport/internal/tlog"
)

type Transport struct {
	name string
	addr string
	mu   sync.Mutex
	conn net.Conn
	ok   atomic.Bool
}

func New(name string, cfg map[string]any) (transport.Transport, error) {
	addr := cfgutil.String(cfg, "addr", "")
	if addr == "" {
		addr = cfgutil.String(cfg, "endpoint", "")
	}
	if addr == "" {
		return nil, fmt.Errorf("tcp transport: addr is required")
	}
	if name == "" {
		name = "tcp"
	}
	return &Transport{name: name, addr: addr}, nil
}

func (t *Transport) Name() string         { return t.name }
func (t *Transport) Type() transport.Type { return transport.TypeTCP }
func (t *Transport) Healthy() bool        { return t.ok.Load() }

func (t *Transport) Start(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", t.addr)
	if err != nil {
		return err
	}
	t.mu.Lock()
	t.conn = conn
	t.mu.Unlock()
	t.ok.Store(true)
	tlog.L().Info("transport started", "name", t.name, "type", string(transport.TypeTCP), "addr", t.addr)
	return nil
}

func (t *Transport) ensure(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.conn != nil {
		return nil
	}
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", t.addr)
	if err != nil {
		t.ok.Store(false)
		return err
	}
	t.conn = conn
	t.ok.Store(true)
	return nil
}

func (t *Transport) Send(ctx context.Context, payload []byte) error {
	if err := t.ensure(ctx); err != nil {
		return err
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := framer.WriteLengthPrefixed(t.conn, payload); err != nil {
		_ = t.conn.Close()
		t.conn = nil
		t.ok.Store(false)
		return err
	}
	t.ok.Store(true)
	return nil
}

func (t *Transport) SendBatch(ctx context.Context, payloads [][]byte) error {
	for _, p := range payloads {
		if err := t.Send(ctx, p); err != nil {
			return err
		}
	}
	return nil
}

func (t *Transport) Flush(context.Context) error { return nil }

func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.ok.Store(false)
	if t.conn != nil {
		err := t.conn.Close()
		t.conn = nil
		return err
	}
	return nil
}

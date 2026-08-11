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

package ws

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/pkg/transport/internal/tlog"
)

type Transport struct {
	name  string
	url   string
	token string
	mu    sync.Mutex
	conn  *websocket.Conn
	ok    atomic.Bool
}

func New(name string, cfg map[string]any) (transport.Transport, error) {
	url := cfgutil.String(cfg, "url", "")
	if url == "" {
		url = cfgutil.String(cfg, "endpoint", "")
	}
	if url == "" {
		return nil, fmt.Errorf("websocket transport: url is required")
	}
	if name == "" {
		name = "websocket"
	}
	return &Transport{
		name:  name,
		url:   url,
		token: cfgutil.String(cfg, "token", ""),
	}, nil
}

func (t *Transport) Name() string         { return t.name }
func (t *Transport) Type() transport.Type { return transport.TypeWebSocket }
func (t *Transport) Healthy() bool        { return t.ok.Load() }

func (t *Transport) Start(ctx context.Context) error {
	if err := t.dial(ctx); err != nil {
		return err
	}
	tlog.L().Info("transport started", "name", t.name, "type", string(transport.TypeWebSocket), "addr", t.url)
	return nil
}

func (t *Transport) dial(ctx context.Context) error {
	header := http.Header{}
	if t.token != "" {
		header.Set("Authorization", "Bearer "+t.token)
	}
	conn, _, err := websocket.Dial(ctx, t.url, &websocket.DialOptions{HTTPHeader: header})
	if err != nil {
		t.ok.Store(false)
		return err
	}
	conn.SetReadLimit(8 << 20)
	t.mu.Lock()
	if t.conn != nil {
		_ = t.conn.Close(websocket.StatusNormalClosure, "")
	}
	t.conn = conn
	t.mu.Unlock()
	t.ok.Store(true)
	return nil
}

func (t *Transport) Send(ctx context.Context, payload []byte) error {
	t.mu.Lock()
	conn := t.conn
	t.mu.Unlock()
	if conn == nil {
		if err := t.dial(ctx); err != nil {
			return err
		}
		t.mu.Lock()
		conn = t.conn
		t.mu.Unlock()
	}
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := conn.Write(writeCtx, websocket.MessageText, payload); err != nil {
		t.ok.Store(false)
		t.mu.Lock()
		_ = conn.Close(websocket.StatusInternalError, "write failed")
		t.conn = nil
		t.mu.Unlock()
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
		err := t.conn.Close(websocket.StatusNormalClosure, "")
		t.conn = nil
		return err
	}
	return nil
}

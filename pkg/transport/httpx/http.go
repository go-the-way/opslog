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

package httpx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/pkg/transport/internal/tlog"
)

type Transport struct {
	name      string
	url       string
	token     string
	client    *http.Client
	ok      atomic.Bool
	firstOK atomic.Bool
}

func New(name string, cfg map[string]any) (transport.Transport, error) {
	url := cfgutil.String(cfg, "url", "")
	if url == "" {
		url = cfgutil.String(cfg, "endpoint", "")
	}
	if url == "" {
		return nil, fmt.Errorf("http transport: url is required")
	}
	if name == "" {
		name = "http"
	}
	timeout := cfgutil.Duration(cfg, "timeout", 5*time.Second)
	return &Transport{
		name:  name,
		url:   url,
		token: cfgutil.String(cfg, "token", ""),
		client: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

func (t *Transport) Name() string         { return t.name }
func (t *Transport) Type() transport.Type { return transport.TypeHTTP }
func (t *Transport) Healthy() bool        { return t.ok.Load() }

func (t *Transport) Start(context.Context) error {
	tlog.L().Info("http transport connecting", "name", t.name, "url", t.url)
	t.ok.Store(true)
	tlog.L().Info("http transport connected", "name", t.name, "url", t.url, "type", string(transport.TypeHTTP))
	return nil
}

func (t *Transport) Send(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.token != "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		t.ok.Store(false)
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		t.ok.Store(false)
		tlog.L().Warn("http transport send failed",
			"name", t.name,
			"url", t.url,
			"status", resp.StatusCode,
			"payload", len(payload),
		)
		return fmt.Errorf("http transport: status %d", resp.StatusCode)
	}
	t.ok.Store(true)
	if t.firstOK.CompareAndSwap(false, true) {
		tlog.L().Info("http transport send ok",
			"name", t.name,
			"url", t.url,
			"status", resp.StatusCode,
		)
	}
	return nil
}

func (t *Transport) SendBatch(ctx context.Context, payloads [][]byte) error {
	if len(payloads) == 0 {
		return nil
	}
	if len(payloads) == 1 {
		return t.Send(ctx, payloads[0])
	}
	buf := bytes.NewBufferString(`{"events":[`)
	for i, p := range payloads {
		if i > 0 {
			buf.WriteByte(',')
		}
		buf.Write(p)
	}
	buf.WriteString(`]}`)
	return t.Send(ctx, buf.Bytes())
}

func (t *Transport) Flush(context.Context) error { return nil }

func (t *Transport) Close() error {
	t.ok.Store(false)
	t.client.CloseIdleConnections()
	return nil
}

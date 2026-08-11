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
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/server/contract"
	disp "github.com/go-the-way/opslog/server/internal/dispatcher"
)

type WebSocket struct {
	name    string
	listen  string
	path    string
	token   string
	decoder contract.Decoder
	srv     *http.Server
}

func NewWebSocket(name string, cfg map[string]any, decoder contract.Decoder) (contract.Input, error) {
	if decoder == nil {
		return nil, fmt.Errorf("websocket input: decoder required")
	}
	if name == "" {
		name = "websocket"
	}
	return &WebSocket{
		name:    name,
		listen:  cfgutil.String(cfg, "listen", ":8600"),
		path:    cfgutil.String(cfg, "path", "/stream"),
		token:   cfgutil.String(cfg, "token", ""),
		decoder: decoder,
	}, nil
}

func (w *WebSocket) Name() string { return w.name }
func (w *WebSocket) Type() string { return "websocket" }

func (w *WebSocket) Start(ctx context.Context, dispatcher contract.Dispatcher) error {
	mux := http.NewServeMux()
	mux.HandleFunc(w.path, func(rw http.ResponseWriter, r *http.Request) {
		if w.token != "" && r.Header.Get("Authorization") != "Bearer "+w.token {
			http.Error(rw, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := websocket.Accept(rw, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		conn.SetReadLimit(8 << 20)
		ctx := disp.WithMeta(r.Context(), w.name, r.RemoteAddr)
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			sig, err := w.decoder.Decode(r.RemoteAddr, data)
			if err != nil {
				continue
			}
			_ = dispatcher.Dispatch(ctx, sig)
		}
	})
	w.srv = &http.Server{Addr: w.listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = w.srv.ListenAndServe() }()
	go func() {
		<-ctx.Done()
		_ = w.Stop(context.Background())
	}()
	return nil
}

func (w *WebSocket) Stop(ctx context.Context) error {
	if w.srv == nil {
		return nil
	}
	return w.srv.Shutdown(ctx)
}

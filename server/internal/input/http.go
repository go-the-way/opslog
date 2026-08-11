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
	"io"
	"net/http"
	"time"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/codec"
	"github.com/go-the-way/opslog/server/contract"
	disp "github.com/go-the-way/opslog/server/internal/dispatcher"
)

type HTTP struct {
	name    string
	listen  string
	path    string
	token   string
	decoder contract.Decoder
	srv     *http.Server
}

func NewHTTP(name string, cfg map[string]any, decoder contract.Decoder) (contract.Input, error) {
	if decoder == nil {
		return nil, fmt.Errorf("http input: decoder required")
	}
	if name == "" {
		name = "http"
	}
	return &HTTP{
		name:    name,
		listen:  cfgutil.String(cfg, "listen", ":8600"),
		path:    cfgutil.String(cfg, "path", "/ingest"),
		token:   cfgutil.String(cfg, "token", ""),
		decoder: decoder,
	}, nil
}

func (h *HTTP) Name() string { return h.name }
func (h *HTTP) Type() string { return "http" }

func (h *HTTP) Start(ctx context.Context, dispatcher contract.Dispatcher) error {
	mux := http.NewServeMux()
	mux.HandleFunc(h.path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if h.token != "" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+h.token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := disp.WithMeta(r.Context(), h.name, r.RemoteAddr)
		events, err := codec.DecodeBatchJSON(body)
		if err != nil {
			// fallback single via decoder
			sig, err2 := h.decoder.Decode(r.RemoteAddr, body)
			if err2 != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			_ = dispatcher.Dispatch(ctx, sig)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		for _, ev := range events {
			_ = dispatcher.Dispatch(ctx, ev)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	h.srv = &http.Server{Addr: h.listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		_ = h.srv.ListenAndServe()
	}()
	go func() {
		<-ctx.Done()
		_ = h.Stop(context.Background())
	}()
	return nil
}

func (h *HTTP) Stop(ctx context.Context) error {
	if h.srv == nil {
		return nil
	}
	return h.srv.Shutdown(ctx)
}

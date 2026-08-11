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

package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/codec"
	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
	"github.com/go-the-way/opslog/server/internal/api"
	"github.com/go-the-way/opslog/server/internal/bootstrap"
	"github.com/go-the-way/opslog/server/internal/config"
	"github.com/go-the-way/opslog/server/internal/dispatcher"
	"github.com/go-the-way/opslog/server/internal/input"
	"github.com/go-the-way/opslog/server/internal/live"
	"github.com/go-the-way/opslog/server/internal/pipeline"
	"github.com/go-the-way/opslog/server/internal/web"
)

type App struct {
	cfg      *config.File
	inputs   []contract.Input
	outs     []contract.Output
	disp     *dispatcher.Async
	liveDisp contract.Dispatcher
	hub      *live.Hub
	http     *http.Server
	log      *slog.Logger
}

func New(cfgPath string) (*App, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}
	reg := bootstrap.NewRegistry()
	log := slog.Default()

	var outs []contract.Output
	for _, raw := range cfg.Outputs {
		comp, err := config.ParseComponent(raw)
		if err != nil {
			return nil, err
		}
		out, err := reg.BuildOutput(contract.OutputType(comp.Type), comp.Name, comp.Cfg)
		if err != nil {
			return nil, err
		}
		outs = append(outs, out)
	}

	var procs []contract.Processor
	if len(cfg.Pipeline) == 0 {
		p, _ := reg.BuildProcessor("normalize", "normalize", nil)
		procs = append(procs, p)
	} else {
		for _, raw := range cfg.Pipeline {
			comp, err := config.ParseComponent(raw)
			if err != nil {
				return nil, err
			}
			p, err := reg.BuildProcessor(comp.Type, comp.Name, comp.Cfg)
			if err != nil {
				return nil, err
			}
			procs = append(procs, p)
		}
	}

	pipe := pipeline.New("main", procs...)
	hub := live.NewHub()
	disp := dispatcher.New(pipe, outs, dispatcher.Options{Logger: log})
	liveDisp := &publishingDispatcher{inner: disp, hub: hub}

	decJSON, _ := reg.BuildDecoder("json", "json", nil)
	decPlain, _ := reg.BuildDecoder("plain", "plain", nil)

	var inputs []contract.Input
	for _, raw := range cfg.Inputs {
		comp, err := config.ParseComponent(raw)
		if err != nil {
			return nil, err
		}
		format := cfgutil.String(comp.Cfg, "format", "json")
		dec := decJSON
		if format == "plain" {
			dec = decPlain
		}
		switch comp.Type {
		case "udp":
			in, err := input.NewUDP(comp.Name, comp.Cfg, dec)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, in)
		case "tcp":
			in, err := input.NewTCP(comp.Name, comp.Cfg, dec)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, in)
		case "grpc":
			in, err := input.NewGRPC(comp.Name, comp.Cfg, dec)
			if err != nil {
				return nil, err
			}
			inputs = append(inputs, in)
		case "http", "websocket":
			// Shared HTTP server below provides /ingest and /stream.
			continue
		default:
			return nil, fmt.Errorf("unsupported input type %q", comp.Type)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/ingest", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if cfg.HTTP.Token != "" && r.Header.Get("Authorization") != "Bearer "+cfg.HTTP.Token {
			log.Warn("ingest unauthorized",
				"input", "http",
				"remote", r.RemoteAddr,
				"path", r.URL.Path,
				"status", http.StatusUnauthorized,
			)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		body, err := io.ReadAll(io.LimitReader(r.Body, 8<<20))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		ctx := dispatcher.WithMeta(r.Context(), "http", r.RemoteAddr)
		events, err := codec.DecodeBatchJSON(body)
		if err != nil {
			sig, err2 := decJSON.Decode(r.RemoteAddr, body)
			if err2 != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			log.Info("ingest accepted",
				"input", "http",
				"remote", r.RemoteAddr,
				"count", 1,
				"kind", string(sig.Kind()),
				"level", sig.Level(),
				"service", sig.Service(),
				"msg", truncateForLog(sig.Message(), 120),
			)
			_ = liveDisp.Dispatch(ctx, sig)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		svc, kind, level, msg := "", "", "", ""
		if n := len(events); n > 0 && events[0] != nil {
			svc = events[0].Service()
			kind = string(events[0].Kind())
			level = events[0].Level()
			msg = truncateForLog(events[0].Message(), 120)
		}
		log.Info("ingest accepted",
			"input", "http",
			"remote", r.RemoteAddr,
			"count", len(events),
			"kind", kind,
			"level", level,
			"service", svc,
			"msg", msg,
		)
		for _, ev := range events {
			if ev != nil && ev.HostValue == "" {
				ev.HostValue = remoteHost(r.RemoteAddr)
			}
			_ = liveDisp.Dispatch(ctx, ev)
		}
		w.WriteHeader(http.StatusAccepted)
	})
	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		if cfg.HTTP.Token != "" && r.Header.Get("Authorization") != "Bearer "+cfg.HTTP.Token {
			log.Warn("ingest unauthorized",
				"input", "websocket",
				"remote", r.RemoteAddr,
				"path", r.URL.Path,
				"status", http.StatusUnauthorized,
			)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		defer conn.Close(websocket.StatusNormalClosure, "")
		conn.SetReadLimit(8 << 20)
		log.Info("websocket client connected", "input", "websocket", "remote", r.RemoteAddr)
		ctx := dispatcher.WithMeta(r.Context(), "websocket", r.RemoteAddr)
		for {
			_, data, err := conn.Read(r.Context())
			if err != nil {
				return
			}
			sig, err := decJSON.Decode(r.RemoteAddr, data)
			if err != nil {
				continue
			}
			log.Info("ingest accepted",
				"input", "websocket",
				"remote", r.RemoteAddr,
				"count", 1,
				"kind", string(sig.Kind()),
				"level", sig.Level(),
				"service", sig.Service(),
				"msg", truncateForLog(sig.Message(), 120),
			)
			_ = liveDisp.Dispatch(ctx, sig)
		}
	})

	apiSrv := &api.Server{Hub: hub}
	for _, out := range outs {
		if q, ok := out.(contract.Queryable); ok && apiSrv.Queryable == nil {
			apiSrv.Queryable = q
		}
		if ar, ok := out.(contract.Archiver); ok && apiSrv.Archiver == nil {
			apiSrv.Archiver = ar
		}
		if rs, ok := out.(contract.Restorer); ok && apiSrv.Restorer == nil {
			apiSrv.Restorer = rs
		}
	}
	apiSrv.Mount(mux)

	// Basic Auth only wraps the static console. /api/*, /ingest, /stream stay open.
	ui := web.Handler()
	if cfg.Web.BasicAuth.Active() {
		ui = web.BasicAuth(cfg.Web.BasicAuth.Username, cfg.Web.BasicAuth.Password, ui)
	}
	mux.Handle("/", ui)

	return &App{
		cfg:      cfg,
		inputs:   inputs,
		outs:     outs,
		disp:     disp,
		liveDisp: liveDisp,
		hub:      hub,
		http:     &http.Server{Addr: cfg.HTTP.Listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second},
		log:      log,
	}, nil
}

func (a *App) Start(ctx context.Context) error {
	a.disp.Start(ctx)
	for _, in := range a.inputs {
		if err := in.Start(ctx, a.liveDisp); err != nil {
			return fmt.Errorf("start input %s: %w", in.Name(), err)
		}
		a.log.Info("input started", "name", in.Name(), "type", in.Type())
	}
	ln, err := net.Listen("tcp", a.cfg.HTTP.Listen)
	if err != nil {
		return fmt.Errorf("http listen %s: %w", a.cfg.HTTP.Listen, err)
	}
	uiURL := webUIAccessURL(a.cfg.HTTP.Listen)
	a.log.Info("http ingest ready", "listen", a.cfg.HTTP.Listen, "path", "/ingest", "token_required", a.cfg.HTTP.Token != "")
	if a.cfg.Web.BasicAuth.Active() {
		a.log.Info("Web UI: "+uiURL, "listen", a.cfg.HTTP.Listen, "basic_auth", true, "username", a.cfg.Web.BasicAuth.Username)
	} else {
		a.log.Info("Web UI: "+uiURL, "listen", a.cfg.HTTP.Listen, "basic_auth", false)
	}
	go func() {
		if err := a.http.Serve(ln); err != nil && err != http.ErrServerClosed {
			a.log.Error("http server failed", "err", err)
		}
	}()
	return nil
}

func remoteHost(addr string) string {
	if addr == "" {
		return ""
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}

func truncateForLog(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// webUIAccessURL builds a browser-usable URL from the configured HTTP listen address.
// Wildcard binds (empty host, 0.0.0.0, ::) are shown as 127.0.0.1.
func webUIAccessURL(listen string) string {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return "http://127.0.0.1:8600/"
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, port) + "/"
}

func (a *App) Stop(ctx context.Context) error {
	_ = a.http.Shutdown(ctx)
	for _, in := range a.inputs {
		_ = in.Stop(ctx)
	}
	_ = a.disp.Stop(ctx)
	for _, out := range a.outs {
		_ = out.Close(ctx)
	}
	return nil
}

type publishingDispatcher struct {
	inner contract.Dispatcher
	hub   *live.Hub
}

func (p *publishingDispatcher) Dispatch(ctx context.Context, sig signal.Signal) error {
	if err := p.inner.Dispatch(ctx, sig); err != nil {
		return err
	}
	p.hub.Publish(sig)
	return nil
}

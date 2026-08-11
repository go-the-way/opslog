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

package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/agent"
	"github.com/go-the-way/opslog/sdk/middleware/opslog4gin"
	"github.com/go-the-way/opslog/sdk/middleware/opslog4proc"
	"github.com/go-the-way/opslog/sdk/middleware/panicopt"
	"github.com/go-the-way/opslog/sdk/policy"
	"github.com/go-the-way/opslog/sdk/transport"
)

var (
	once   sync.Once
	mu     sync.RWMutex
	ag     sdk.Agent
	logger sdk.Logger
)

// MustInitFromEnv loads ConfigFromEnv, applies opts, and panics only on programmer
// misuse after Enable=true (transport/agent construction). Env-disabled is not an error.
func MustInitFromEnv(opts ...Option) {
	if err := Init(ConfigFromEnv(opts...)); err != nil {
		panic(err)
	}
}

// MustInit is like Init but panics on error.
func MustInit(cfg Config) {
	if err := Init(cfg); err != nil {
		panic(err)
	}
}

// Init starts a process-wide Agent (idempotent). No-op when Skip or !Enable.
func Init(cfg Config) error {
	var initErr error
	once.Do(func() {
		if cfg.Skip || !cfg.Enable {
			return
		}
		a, lg, err := startAgent(cfg)
		if err != nil {
			initErr = err
			return
		}
		mu.Lock()
		ag = a
		logger = lg
		mu.Unlock()
		if cfg.bridgeEnabled() {
			bridgeOutputs(lg)
		}
		// Lifecycle lines go to stderr from agent.Start / HTTP transport Start
		// (not via lg.Info, which would only enqueue OpsLog signals).
	})
	return initErr
}

func startAgent(cfg Config) (sdk.Agent, sdk.Logger, error) {
	if cfg.Service == "" {
		cfg.Service = "app"
	}
	if cfg.Level == "" {
		cfg.Level = "debug"
	}
	if cfg.Transport == "" {
		cfg.Transport = "http"
	}
	if cfg.HTTPURL == "" {
		cfg.HTTPURL = "http://127.0.0.1:8600/ingest"
	}

	opts := []agent.Option{
		agent.WithService(cfg.Service),
		agent.WithLevel(cfg.Level),
	}
	if cfg.Version != "" {
		opts = append(opts, agent.WithVersion(cfg.Version))
	}

	wantHTTP, wantUDP := parseTransport(cfg.Transport)
	var hasHTTP, hasUDP bool
	if wantHTTP {
		t, err := transport.NewHTTPTransport("http", cfg.HTTPURL, cfg.HTTPToken)
		if err != nil {
			return nil, nil, fmt.Errorf("sdk/integration: http transport: %w", err)
		}
		opts = append(opts, agent.WithTransport(t))
		hasHTTP = true
	}
	if wantUDP && cfg.UDPEndpoint != "" {
		t, err := transport.NewUDPTransport("udp", cfg.UDPEndpoint)
		if err != nil {
			return nil, nil, fmt.Errorf("sdk/integration: udp transport: %w", err)
		}
		opts = append(opts, agent.WithTransport(t))
		hasUDP = true
	}
	if !hasHTTP && !hasUDP {
		return nil, nil, fmt.Errorf("sdk/integration: no transport configured")
	}
	switch {
	case hasUDP && hasHTTP:
		opts = append(opts, agent.WithPolicy(policy.NewLevelPolicy("udp", "http")))
	case hasUDP:
		opts = append(opts, agent.WithPolicy(policy.NewLevelPolicy("udp", "udp")))
	default:
		opts = append(opts, agent.WithPolicy(policy.NewLevelPolicy("http", "http")))
	}

	a, err := agent.NewAgent(opts...)
	if err != nil {
		return nil, nil, err
	}
	if err := a.Start(context.Background()); err != nil {
		_ = a.Close()
		return nil, nil, err
	}
	return a, a.Logger(), nil
}

// Enabled reports whether the process-wide Agent is running.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return ag != nil
}

// Agent returns the process-wide Agent, or nil when disabled.
func Agent() sdk.Agent {
	mu.RLock()
	defer mu.RUnlock()
	return ag
}

// Logger returns the process-wide Logger, or nil when disabled.
func Logger() sdk.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return logger
}

// Close syncs and closes the process-wide Agent (safe when disabled).
func Close() {
	mu.Lock()
	a := ag
	ag = nil
	logger = nil
	mu.Unlock()
	if a == nil {
		return
	}
	_ = a.Sync()
	_ = a.Close()
}

// GinRecovery returns opslog4gin.Recovery when the Agent is enabled; otherwise gin.Recovery.
func GinRecovery(opts ...panicopt.Option) gin.HandlerFunc {
	lg := Logger()
	if lg == nil {
		return gin.Recovery()
	}
	if len(opts) == 0 {
		opts = []panicopt.Option{panicopt.WithAttrs(sdk.String("component", "http"))}
	}
	return opslog4gin.Recovery(lg, opts...)
}

// Guard returns a defer-friendly panic guard; no-op when the Agent is disabled.
func Guard(opts ...panicopt.Option) func() {
	lg := Logger()
	if lg == nil {
		return func() {}
	}
	if len(opts) == 0 {
		opts = []panicopt.Option{panicopt.WithAttrs(sdk.String("component", "process"))}
	}
	return opslog4proc.Guard(lg, opts...)
}

// Main runs fn with Guard + Close around it. Call Init/MustInitFromEnv before Main
// (or pass a cfg to MainWithConfig).
func Main(run func() error) error {
	defer Guard()()
	defer Close()
	if run == nil {
		return nil
	}
	return run()
}

// MainWithConfig initializes from cfg then runs Main.
func MainWithConfig(cfg Config, run func() error) error {
	if err := Init(cfg); err != nil {
		return err
	}
	return Main(run)
}

func bridgeOutputs(lg sdk.Logger) {
	if lg == nil {
		return
	}
	w := &loggerWriter{logger: lg, source: "stdlib"}
	log.SetOutput(io.MultiWriter(os.Stderr, w))
	gin.DefaultWriter = io.MultiWriter(os.Stdout, &loggerWriter{logger: lg, source: "gin"})
	gin.DefaultErrorWriter = io.MultiWriter(os.Stderr, &loggerWriter{logger: lg, source: "gin", level: "error"})
}

type loggerWriter struct {
	logger sdk.Logger
	source string
	level  string
}

func (w *loggerWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\r\n")
	if msg == "" || w.logger == nil {
		return len(p), nil
	}
	attrs := []sdk.Attr{sdk.String("source", w.source)}
	switch strings.ToLower(w.level) {
	case "error", "warn":
		w.logger.Error(msg, attrs...)
	default:
		w.logger.Info(msg, attrs...)
	}
	return len(p), nil
}

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

package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/enricher"
	"github.com/go-the-way/opslog/sdk/policy"
	sdktransport "github.com/go-the-way/opslog/sdk/transport"
)

type agentImpl struct {
	service    string
	host       string
	minLevel   string
	formatter  sdk.Formatter
	policy     sdk.Policy
	hooks      []sdk.Hook
	collectors []sdk.Collector
	transports []transport.Transport
	interval   time.Duration

	ch       chan signal.Signal
	wg       sync.WaitGroup
	closed   atomic.Bool
	cancel   context.CancelFunc
	mu       sync.Mutex
	dropped  atomic.Uint64
	inflight atomic.Int64
}

// NewAgent builds an embeddable OpsLog agent.
func NewAgent(opts ...Option) (sdk.Agent, error) {
	o := defaultOptions()
	for _, opt := range opts {
		opt(o)
	}
	if o.service == "" {
		o.service = "app"
	}
	if o.host == "" {
		h, _ := os.Hostname()
		o.host = h
	}
	if o.endpoint != "" {
		url := normalizeHTTPIngestURL(o.endpoint)
		t, err := sdktransport.NewHTTPTransport("http", url, o.token)
		if err != nil {
			return nil, fmt.Errorf("sdk/agent: http transport: %w", err)
		}
		o.transports = append(o.transports, t)
		if o.policy == nil {
			o.policy = policy.NewLevelPolicy("http", "http")
		}
	}
	if len(o.transports) == 0 {
		return nil, fmt.Errorf("sdk/agent: at least one transport is required (use WithEndpoint or WithTransport)")
	}
	if o.policy == nil {
		o.policy = policy.NewLevelPolicy(o.transports[0].Name(), o.transports[0].Name())
	}
	if o.queueSize <= 0 {
		o.queueSize = 2048
	}
	hooks := o.hooks
	if o.diagnosticEnabled {
		diagOpts := append([]enricher.Option(nil), o.diagnosticOpts...)
		diagOpts = append(diagOpts,
			enricher.WithService(o.service),
		)
		if o.version != "" {
			diagOpts = append(diagOpts, enricher.WithVersion(o.version))
		}
		if o.gitSHA != "" {
			diagOpts = append(diagOpts, enricher.WithGitSHA(o.gitSHA))
		}
		hooks = append([]sdk.Hook{enricher.NewDiagnostic(diagOpts...)}, hooks...)
	}
	a := &agentImpl{
		service:    o.service,
		host:       o.host,
		minLevel:   o.minLevel,
		formatter:  o.formatter,
		policy:     o.policy,
		hooks:      hooks,
		collectors: o.collectors,
		transports: o.transports,
		interval:   o.interval,
		ch:         make(chan signal.Signal, o.queueSize),
	}
	return a, nil
}

func (a *agentImpl) Logger() sdk.Logger {
	return &loggerImpl{agent: a, closed: &a.closed}
}

func (a *agentImpl) Transports() []transport.Transport { return a.transports }
func (a *agentImpl) Collectors() []sdk.Collector       { return a.collectors }

func (a *agentImpl) Start(ctx context.Context) error {
	names := make([]string, 0, len(a.transports))
	for _, t := range a.transports {
		if err := t.Start(ctx); err != nil {
			return fmt.Errorf("start transport %s: %w", t.Name(), err)
		}
		endpoint := t.Name() + "/" + string(t.Type())
		names = append(names, endpoint)
		agentLog().Info("agent transport ready",
			"service", a.service,
			"name", t.Name(),
			"type", string(t.Type()),
		)
	}
	cctx, cancel := context.WithCancel(ctx)
	a.cancel = cancel
	workers := 2
	a.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go a.worker(cctx)
	}
	if len(a.collectors) > 0 {
		a.wg.Add(1)
		go a.collectLoop(cctx)
	}
	agentLog().Info("agent started", "service", a.service, "transports", strings.Join(names, ","))
	return nil
}

func (a *agentImpl) enqueue(sig signal.Signal) {
	if a.closed.Load() {
		return
	}
	select {
	case a.ch <- sig:
	default:
		n := a.dropped.Add(1)
		if n == 1 || n%100 == 0 {
			agentLog().Warn("queue full, dropping signal", "dropped", n, "service", a.service)
		}
	}
}

func (a *agentImpl) worker(ctx context.Context) {
	defer a.wg.Done()
	for {
		select {
		case <-ctx.Done():
			// Drain with a live context so HTTP ingest is not cancelled mid-flight.
			drainCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			for {
				select {
				case sig := <-a.ch:
					_ = a.sendOne(drainCtx, sig)
				default:
					cancel()
					return
				}
			}
		case sig := <-a.ch:
			_ = a.sendOne(ctx, sig)
		}
	}
}

func (a *agentImpl) sendOne(ctx context.Context, sig signal.Signal) error {
	a.inflight.Add(1)
	defer a.inflight.Add(-1)

	cur := sig
	for _, h := range a.hooks {
		out, keep := h.BeforeSend(cur)
		if !keep {
			return nil
		}
		cur = out
	}
	payload, err := a.formatter.Format(cur)
	if err != nil {
		agentLog().Warn("format failed", "service", a.service, "level", cur.Level(), "err", err)
		return err
	}
	targets := a.policy.Select(cur, a.transports)
	if len(targets) == 0 {
		agentLog().Warn("no transport selected", "service", a.service, "level", cur.Level(), "payload", len(payload))
		return fmt.Errorf("sdk/agent: no transport selected")
	}
	var first error
	for _, t := range targets {
		if err := t.Send(ctx, payload); err != nil {
			agentLog().Warn("send failed",
				"transport", t.Name(),
				"service", a.service,
				"level", cur.Level(),
				"payload", len(payload),
				"err", err,
			)
			if first == nil {
				first = err
			}
			// Error/fatal/panic: if preferred transport fails, try any other healthy transport.
			if isReliableLevel(cur.Level()) {
				if fb := a.fallbackSend(ctx, t, payload); fb == nil {
					first = nil
				}
			}
			continue
		}
	}
	return first
}

func isReliableLevel(level string) bool {
	switch strings.ToLower(level) {
	case "error", "fatal", "panic":
		return true
	default:
		return false
	}
}

func (a *agentImpl) fallbackSend(ctx context.Context, failed transport.Transport, payload []byte) error {
	var first error
	for _, t := range a.transports {
		if t == nil || t == failed {
			continue
		}
		if err := t.Send(ctx, payload); err != nil {
			agentLog().Warn("fallback send failed",
				"transport", t.Name(),
				"service", a.service,
				"payload", len(payload),
				"err", err,
			)
			if first == nil {
				first = err
			}
			continue
		}
		return nil
	}
	return first
}

func (a *agentImpl) collectLoop(ctx context.Context) {
	defer a.wg.Done()
	ticker := time.NewTicker(a.interval)
	defer ticker.Stop()
	a.runCollectors(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runCollectors(ctx)
		}
	}
}

func (a *agentImpl) runCollectors(ctx context.Context) {
	for _, c := range a.collectors {
		sigs, err := c.Collect(ctx)
		if err != nil {
			continue
		}
		for _, s := range sigs {
			if ev, ok := s.(*signal.Event); ok {
				if ev.ServiceValue == "" {
					ev.ServiceValue = a.service
				}
				if ev.HostValue == "" {
					ev.HostValue = a.host
				}
			}
			a.enqueue(s)
		}
	}
}

func (a *agentImpl) Sync() error {
	deadline := time.After(3 * time.Second)
	for {
		if len(a.ch) == 0 && a.inflight.Load() == 0 {
			break
		}
		select {
		case <-deadline:
			goto flush
		case <-time.After(20 * time.Millisecond):
		}
	}
flush:
	a.mu.Lock()
	defer a.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, t := range a.transports {
		if err := t.Flush(ctx); err != nil {
			agentLog().Warn("flush failed", "transport", t.Name(), "service", a.service, "err", err)
		}
	}
	return nil
}

func (a *agentImpl) Close() error {
	if a.closed.Swap(true) {
		return nil
	}
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
	_ = a.Sync()
	var first error
	for _, t := range a.transports {
		if err := t.Close(); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (a *agentImpl) Dropped() uint64 { return a.dropped.Load() }

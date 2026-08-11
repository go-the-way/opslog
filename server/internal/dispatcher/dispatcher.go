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

package dispatcher

import (
	"context"
	"log/slog"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
)

type metaKey struct{}

// Meta carries ingest provenance for receive logging.
type Meta struct {
	Input  string
	Remote string
}

// WithMeta attaches input name and remote address to ctx for Dispatch logging.
func WithMeta(ctx context.Context, input, remote string) context.Context {
	return context.WithValue(ctx, metaKey{}, Meta{Input: input, Remote: remote})
}

func metaFrom(ctx context.Context) Meta {
	m, _ := ctx.Value(metaKey{}).(Meta)
	return m
}

// Async dispatches signals through a pipeline into outputs with batching.
type Async struct {
	pipeline contract.Pipeline
	outputs  []contract.Output

	ch     chan signal.Signal
	batchN int
	flush  time.Duration

	wg     sync.WaitGroup
	cancel context.CancelFunc
	log    *slog.Logger
}

type Options struct {
	QueueSize    int
	BatchSize    int
	FlushEvery   time.Duration
	Logger       *slog.Logger
}

func New(pipeline contract.Pipeline, outputs []contract.Output, opt Options) *Async {
	if opt.QueueSize <= 0 {
		opt.QueueSize = 4096
	}
	if opt.BatchSize <= 0 {
		opt.BatchSize = 64
	}
	if opt.FlushEvery <= 0 {
		opt.FlushEvery = 200 * time.Millisecond
	}
	if opt.Logger == nil {
		opt.Logger = slog.Default()
	}
	return &Async{
		pipeline: pipeline,
		outputs:  outputs,
		ch:       make(chan signal.Signal, opt.QueueSize),
		batchN:   opt.BatchSize,
		flush:    opt.FlushEvery,
		log:      opt.Logger,
	}
}

func (d *Async) Start(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	d.cancel = cancel
	d.wg.Add(1)
	go d.loop(ctx)
}

func (d *Async) Stop(ctx context.Context) error {
	if d.cancel != nil {
		d.cancel()
	}
	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return d.flushAll(ctx)
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (d *Async) Dispatch(ctx context.Context, sig signal.Signal) error {
	if d.pipeline != nil {
		out, keep, err := d.pipeline.Process(ctx, sig)
		if err != nil {
			return err
		}
		if !keep {
			return nil
		}
		sig = out
	}
	select {
	case d.ch <- sig:
		m := metaFrom(ctx)
		d.log.Info("signal received",
			"input", m.Input,
			"remote", m.Remote,
			"kind", sig.Kind(),
			"level", sig.Level(),
			"service", sig.Service(),
			"host", sig.Host(),
			"msg", truncateMsg(sig.Message(), 120),
		)
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		// drop on full queue to protect ingest path
		d.log.Warn("dispatcher queue full, dropping signal", "kind", sig.Kind(), "service", sig.Service())
		return nil
	}
}

func truncateMsg(s string, max int) string {
	if max <= 0 || s == "" || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max]) + "…"
}

func (d *Async) DispatchBatch(ctx context.Context, batch []signal.Signal) error {
	for _, s := range batch {
		if err := d.Dispatch(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func (d *Async) loop(ctx context.Context) {
	defer d.wg.Done()
	ticker := time.NewTicker(d.flush)
	defer ticker.Stop()
	batch := make([]signal.Signal, 0, d.batchN)
	flushBatch := func() {
		if len(batch) == 0 {
			return
		}
		d.write(ctx, batch)
		batch = batch[:0]
	}
	for {
		select {
		case <-ctx.Done():
			// drain
			for {
				select {
				case s := <-d.ch:
					batch = append(batch, s)
					if len(batch) >= d.batchN {
						flushBatch()
					}
				default:
					flushBatch()
					return
				}
			}
		case s := <-d.ch:
			batch = append(batch, s)
			if len(batch) >= d.batchN {
				flushBatch()
			}
		case <-ticker.C:
			flushBatch()
		}
	}
}

func (d *Async) write(ctx context.Context, batch []signal.Signal) {
	for _, out := range d.outputs {
		if err := out.Write(ctx, batch); err != nil {
			d.log.Error("output write failed", "output", out.Name(), "type", out.Type(), "err", err)
		}
	}
}

func (d *Async) flushAll(ctx context.Context) error {
	var first error
	for _, out := range d.outputs {
		if err := out.Flush(ctx); err != nil && first == nil {
			first = err
		}
	}
	return first
}

var _ contract.Dispatcher = (*Async)(nil)
var _ contract.BatchDispatcher = (*Async)(nil)

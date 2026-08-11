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

package collectors

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/sdk"
)

type Target struct {
	Name   string
	Target string // tcp://host:port or http://...
}

type ProbeCollector struct {
	targets []Target
	client  *http.Client
}

func NewProbe(targets ...Target) sdk.Collector {
	return &ProbeCollector{
		targets: targets,
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

func (p *ProbeCollector) Name() string { return "probe" }

func (p *ProbeCollector) Collect(ctx context.Context) ([]signal.Signal, error) {
	out := make([]signal.Signal, 0, len(p.targets))
	for _, t := range p.targets {
		out = append(out, p.check(ctx, t))
	}
	return out, nil
}

func (p *ProbeCollector) check(ctx context.Context, t Target) signal.Signal {
	start := time.Now()
	ok := false
	errMsg := ""
	switch {
	case strings.HasPrefix(t.Target, "tcp://"):
		addr := strings.TrimPrefix(t.Target, "tcp://")
		d := net.Dialer{Timeout: 3 * time.Second}
		c, err := d.DialContext(ctx, "tcp", addr)
		if err != nil {
			errMsg = err.Error()
		} else {
			ok = true
			_ = c.Close()
		}
	case strings.HasPrefix(t.Target, "http://"), strings.HasPrefix(t.Target, "https://"):
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.Target, nil)
		if err != nil {
			errMsg = err.Error()
		} else {
			resp, err := p.client.Do(req)
			if err != nil {
				errMsg = err.Error()
			} else {
				_ = resp.Body.Close()
				ok = resp.StatusCode < 500
				if !ok {
					errMsg = resp.Status
				}
			}
		}
	default:
		errMsg = "unsupported target scheme"
	}
	return &signal.Event{
		KindValue:    signal.KindProbe,
		TimeValue:    time.Now(),
		LevelValue:   map[bool]string{true: "info", false: "error"}[ok],
		MessageValue: t.Name,
		AttrsValue: map[string]any{
			"target":     t.Target,
			"ok":         ok,
			"latency_ms": time.Since(start).Milliseconds(),
			"error":      errMsg,
		},
	}
}

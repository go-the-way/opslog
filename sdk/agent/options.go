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
	"strings"
	"time"

	pkgtransport "github.com/go-the-way/opslog/pkg/transport"
	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/enricher"
	"github.com/go-the-way/opslog/sdk/formatter"
)

type Option func(*options)

type options struct {
	service    string
	host       string
	version    string
	gitSHA     string
	minLevel   string
	queueSize  int
	workers    int
	dropOldest bool
	formatter  sdk.Formatter
	policy     sdk.Policy
	hooks      []sdk.Hook
	collectors []sdk.Collector
	transports []pkgtransport.Transport
	interval   time.Duration

	// endpoint + token are applied in NewAgent so option order does not matter.
	endpoint string
	token    string

	diagnosticEnabled bool
	diagnosticOpts    []enricher.Option
}

func defaultOptions() *options {
	return &options{
		minLevel:          "debug",
		queueSize:         2048,
		workers:           2,
		formatter:         formatter.NewJSONFormatter(),
		interval:          15 * time.Second,
		diagnosticEnabled: true,
	}
}

func WithService(service string) Option {
	return func(o *options) { o.service = service }
}

func WithHost(host string) Option {
	return func(o *options) { o.host = host }
}

// WithVersion sets the diagnostic attrs["version"] (also used by the default enricher).
func WithVersion(version string) Option {
	return func(o *options) { o.version = version }
}

// WithGitSHA sets the diagnostic attrs["git_sha"] (also used by the default enricher).
func WithGitSHA(sha string) Option {
	return func(o *options) { o.gitSHA = sha }
}

func WithLevel(level string) Option {
	return func(o *options) { o.minLevel = level }
}

func WithQueueSize(n int) Option {
	return func(o *options) { o.queueSize = n }
}

func WithFormatter(f sdk.Formatter) Option {
	return func(o *options) { o.formatter = f }
}

func WithPolicy(p sdk.Policy) Option {
	return func(o *options) { o.policy = p }
}

func WithHook(h sdk.Hook) Option {
	return func(o *options) { o.hooks = append(o.hooks, h) }
}

// WithoutDiagnosticEnricher disables the default diagnostic context hook.
func WithoutDiagnosticEnricher() Option {
	return func(o *options) { o.diagnosticEnabled = false }
}

// WithDiagnosticEnricher keeps the default enricher enabled and applies extra options
// (version/git sha/env allow-list/optional filtered environ snapshot).
func WithDiagnosticEnricher(opts ...enricher.Option) Option {
	return func(o *options) {
		o.diagnosticEnabled = true
		o.diagnosticOpts = append(o.diagnosticOpts, opts...)
	}
}

func WithCollector(c sdk.Collector) Option {
	return func(o *options) { o.collectors = append(o.collectors, c) }
}

func WithTransport(t pkgtransport.Transport) Option {
	return func(o *options) { o.transports = append(o.transports, t) }
}

func WithCollectInterval(d time.Duration) Option {
	return func(o *options) { o.interval = d }
}

// WithEndpoint configures an HTTP ingest URL (transport built in NewAgent).
// Prefer HTTP over UDP: diagnostic attrs (environ/sys/stack) often exceed
// OS UDP datagram limits and are silently dropped.
//
// endpoint may be a full URL (http://host:8600/ingest) or host:port
// (expanded to http://host:port/ingest).
// Pair with WithToken when the server sets http.token.
func WithEndpoint(endpoint string) Option {
	return func(o *options) { o.endpoint = endpoint }
}

// WithToken sets Bearer token for the HTTP ingest transport from WithEndpoint
// (Authorization: Bearer <token>). Empty token disables the header.
func WithToken(token string) Option {
	return func(o *options) { o.token = token }
}

func normalizeHTTPIngestURL(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "http://127.0.0.1:8600/ingest"
	}
	if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
		return endpoint
	}
	endpoint = strings.TrimPrefix(endpoint, "tcp://")
	return "http://" + endpoint + "/ingest"
}

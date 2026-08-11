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
	"os"
	"strings"
)

// Environment variable names used by ConfigFromEnv.
const (
	EnvEnable    = "OPSLOG_ENABLE"
	EnvService   = "OPSLOG_SERVICE"
	EnvLevel     = "OPSLOG_LEVEL"
	EnvHttpURL   = "OPSLOG_HTTP_URL"
	EnvHttpToken = "OPSLOG_HTTP_TOKEN" // optional Bearer for HTTP/WS ingest
	EnvTransport = "OPSLOG_TRANSPORT"
	EnvEndpoint  = "OPSLOG_ENDPOINT" // optional UDP; only when transport includes udp
)

// Config controls process-wide Agent bootstrap.
type Config struct {
	// Enable starts the Agent when true. ConfigFromEnv maps OPSLOG_ENABLE (T/true/1).
	Enable bool
	// Skip forces a no-op Init (e.g. version-only CLI).
	Skip bool
	// Service is the reported service name (default "app").
	Service string
	// Version is attached via the default diagnostic enricher.
	Version string
	// Level is the minimum log level (default "debug").
	Level string
	// HTTPURL is the HTTP ingest URL (default http://127.0.0.1:8600/ingest).
	HTTPURL string
	// HTTPToken is sent as Authorization: Bearer <token> when non-empty
	// (must match server configs/opslog.yml http.token).
	HTTPToken string
	// Transport selects transports: "http" (default), "udp", or "http,udp".
	Transport string
	// UDPEndpoint is used only when Transport includes udp (default 127.0.0.1:8140).
	UDPEndpoint string
	// BridgeStdlog redirects standard log and gin DefaultWriter/ErrorWriter into the Agent.
	// Default true when Enable is true.
	BridgeStdlog *bool
}

// Option mutates Config before env overrides are applied in ConfigFromEnv.
type Option func(*Config)

func WithEnable(v bool) Option      { return func(c *Config) { c.Enable = v } }
func WithSkip(v bool) Option        { return func(c *Config) { c.Skip = v } }
func WithService(s string) Option   { return func(c *Config) { c.Service = s } }
func WithVersion(v string) Option   { return func(c *Config) { c.Version = v } }
func WithLevel(level string) Option { return func(c *Config) { c.Level = level } }
func WithHTTPURL(url string) Option { return func(c *Config) { c.HTTPURL = url } }
func WithHTTPToken(token string) Option {
	return func(c *Config) { c.HTTPToken = token }
}
func WithTransport(t string) Option { return func(c *Config) { c.Transport = t } }
func WithUDPEndpoint(addr string) Option {
	return func(c *Config) { c.UDPEndpoint = addr }
}
func WithBridgeStdlog(v bool) Option {
	return func(c *Config) { c.BridgeStdlog = &v }
}

// ConfigFromEnv builds Config as: package defaults → opts → non-empty OPSLOG_* env.
// Default Enable=false; Transport=http; HTTP ingest on :8600.
func ConfigFromEnv(opts ...Option) Config {
	c := Config{
		Enable:      false,
		Service:     "app",
		Level:       "debug",
		HTTPURL:     "http://127.0.0.1:8600/ingest",
		Transport:   "http",
		UDPEndpoint: "127.0.0.1:8140",
	}
	for _, opt := range opts {
		opt(&c)
	}
	if v, ok := os.LookupEnv(EnvEnable); ok {
		c.Enable = envTruthy(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvService)); v != "" {
		c.Service = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvLevel)); v != "" {
		c.Level = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvHttpURL)); v != "" {
		c.HTTPURL = v
	}
	if v, ok := os.LookupEnv(EnvHttpToken); ok {
		c.HTTPToken = strings.TrimSpace(v)
	}
	if v := strings.TrimSpace(os.Getenv(EnvTransport)); v != "" {
		c.Transport = v
	}
	if v := strings.TrimSpace(os.Getenv(EnvEndpoint)); v != "" {
		c.UDPEndpoint = v
	}
	return c
}

func (c Config) bridgeEnabled() bool {
	if c.BridgeStdlog != nil {
		return *c.BridgeStdlog
	}
	return c.Enable
}

func envTruthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseTransport(s string) (httpOn, udpOn bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" || s == "both" || s == "all" {
		return true, false
	}
	for _, p := range strings.Split(s, ",") {
		switch strings.TrimSpace(p) {
		case "http", "https":
			httpOn = true
		case "udp":
			udpOn = true
		}
	}
	if !httpOn && !udpOn {
		return true, false
	}
	return httpOn, udpOn
}

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
	"testing"
)

func TestConfigFromEnvDefaultsAndOverrides(t *testing.T) {
	t.Setenv(EnvEnable, "")
	t.Setenv(EnvService, "")
	t.Setenv(EnvLevel, "")
	t.Setenv(EnvHttpURL, "")
	t.Setenv(EnvHttpToken, "")
	t.Setenv(EnvTransport, "")
	t.Setenv(EnvEndpoint, "")

	cfg := ConfigFromEnv(WithService("cloudsystem-auth"), WithVersion("1.2.3"))
	if cfg.Enable {
		t.Fatalf("enable default: got true")
	}
	if cfg.Service != "cloudsystem-auth" {
		t.Fatalf("service default opt: got %q", cfg.Service)
	}
	if cfg.Version != "1.2.3" {
		t.Fatalf("version: got %q", cfg.Version)
	}
	if cfg.Transport != "http" {
		t.Fatalf("transport default: got %q", cfg.Transport)
	}
	if cfg.HTTPURL != "http://127.0.0.1:8600/ingest" {
		t.Fatalf("http url default: got %q", cfg.HTTPURL)
	}
	if cfg.HTTPToken != "" {
		t.Fatalf("http token default: got %q", cfg.HTTPToken)
	}

	t.Setenv(EnvEnable, "T")
	t.Setenv(EnvService, "from-env")
	t.Setenv(EnvTransport, "udp,http")
	t.Setenv(EnvHttpToken, "change-me")
	cfg = ConfigFromEnv(WithService("cloudsystem-auth"))
	if !cfg.Enable {
		t.Fatalf("enable from env")
	}
	if cfg.Service != "from-env" {
		t.Fatalf("service env override: got %q", cfg.Service)
	}
	if cfg.HTTPToken != "change-me" {
		t.Fatalf("http token env: got %q", cfg.HTTPToken)
	}
	httpOn, udpOn := parseTransport(cfg.Transport)
	if !httpOn || !udpOn {
		t.Fatalf("transport parse: http=%v udp=%v", httpOn, udpOn)
	}
}

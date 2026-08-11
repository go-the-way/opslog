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

package enricher

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
)

func TestIsSecretEnvKey(t *testing.T) {
	secrets := []string{"DB_PASSWORD", "API_TOKEN", "MY_SECRET", "PRIVATE_KEY", "AWS_ACCESS_KEY_ID", "AUTH_HEADER"}
	for _, k := range secrets {
		if !IsSecretEnvKey(k) {
			t.Fatalf("expected secret: %s", k)
		}
	}
	safe := []string{"APP_ENV", "PROFILE", "VERSION", "HOSTNAME", "POD_NAME", "GO_ENV"}
	for _, k := range safe {
		if IsSecretEnvKey(k) {
			t.Fatalf("expected safe: %s", k)
		}
	}
}

func TestLookupProfileOrder(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("ENV", "")
	t.Setenv("GO_ENV", "")
	t.Setenv("PROFILE", "staging")
	_ = os.Unsetenv("APP_ENV")
	_ = os.Unsetenv("ENV")
	_ = os.Unsetenv("GO_ENV")
	t.Setenv("PROFILE", "staging")

	key, val := LookupProfile()
	if key != "PROFILE" || val != "staging" {
		t.Fatalf("got %s=%s", key, val)
	}

	t.Setenv("APP_ENV", "prod")
	key, val = LookupProfile()
	if key != "APP_ENV" || val != "prod" {
		t.Fatalf("got %s=%s want APP_ENV=prod", key, val)
	}
}

func TestCollectAllowListedEnvSkipsSecrets(t *testing.T) {
	t.Setenv("APP_ENV", "dev")
	t.Setenv("DB_PASSWORD", "nope")
	got := CollectAllowListedEnv([]string{"APP_ENV", "DB_PASSWORD"})
	if got["APP_ENV"] != "dev" {
		t.Fatalf("missing APP_ENV: %#v", got)
	}
	if _, ok := got["DB_PASSWORD"]; ok {
		t.Fatalf("secret leaked: %#v", got)
	}
}

func TestDiagnosticBeforeSend(t *testing.T) {
	t.Setenv("APP_ENV", "test")
	t.Setenv("GIT_SHA", "abc123")

	h := NewDiagnostic(WithVersion("1.2.3"), WithService("demo"))
	in := &signal.Event{
		KindValue:    signal.KindLog,
		TimeValue:    time.Now(),
		LevelValue:   "error",
		ServiceValue: "demo",
		MessageValue: "panic recovered: boom",
		AttrsValue: map[string]any{
			"panic":     "boom",
			"stack":     "goroutine 1...",
			"recovered": true,
		},
	}
	out, keep := h.BeforeSend(in)
	if !keep {
		t.Fatal("dropped")
	}
	ev := out.(*signal.Event)
	a := ev.AttrsValue
	if a["panic"] != "boom" {
		t.Fatalf("overwrote panic: %#v", a)
	}
	if a["profile"] != "test" {
		t.Fatalf("profile=%v", a["profile"])
	}
	if a["version"] != "1.2.3" {
		t.Fatalf("version=%v", a["version"])
	}
	if a["git_sha"] != "abc123" {
		t.Fatalf("git_sha=%v", a["git_sha"])
	}
	if a["go_version"] != runtime.Version() {
		t.Fatalf("go_version=%v", a["go_version"])
	}
	if a["goos"] != runtime.GOOS || a["goarch"] != runtime.GOARCH {
		t.Fatalf("goos/goarch missing")
	}
	if _, ok := a["pid"].(int); !ok {
		t.Fatalf("pid type %T", a["pid"])
	}
	if a["cwd"] == "" || a["hostname"] == "" {
		t.Fatalf("cwd/hostname missing: %#v", a)
	}
	env, _ := a["env"].(map[string]any)
	if env["APP_ENV"] != "test" {
		t.Fatalf("env map: %#v", env)
	}
	environ, _ := a["environ"].(map[string]any)
	if environ["APP_ENV"] != "test" {
		t.Fatalf("environ map missing APP_ENV: %#v", environ)
	}
	proc, _ := a["process_environ"].(map[string]any)
	if proc["APP_ENV"] != "test" {
		t.Fatalf("process_environ missing APP_ENV: %#v", proc)
	}
	startup, _ := a["startup_environ"].(map[string]any)
	if startup["APP_ENV"] != "test" {
		t.Fatalf("startup_environ missing APP_ENV: %#v", startup)
	}
	if _, ok := a["system_environ"].(map[string]any); !ok {
		t.Fatalf("system_environ missing: %#v", a["system_environ"])
	}
	sys, _ := a["sys"].(map[string]any)
	if sys["num_cpu"] == nil || sys["goroutines"] == nil {
		t.Fatalf("sys missing: %#v", sys)
	}
	// original must be unchanged
	if _, ok := in.AttrsValue["profile"]; ok {
		t.Fatal("mutated input")
	}
}

func TestDiagnosticDoesNotOverwrite(t *testing.T) {
	h := NewDiagnostic(WithVersion("from-hook"))
	in := &signal.Event{
		KindValue: signal.KindLog,
		TimeValue: time.Now(),
		AttrsValue: map[string]any{
			"version": "caller",
		},
	}
	out, _ := h.BeforeSend(in)
	if out.(*signal.Event).AttrsValue["version"] != "caller" {
		t.Fatal("should keep caller version")
	}
}

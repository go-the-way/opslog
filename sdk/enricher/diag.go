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
	"sync"

	"github.com/go-the-way/opslog/pkg/signal"
)

// Option configures a Diagnostic hook.
type Option func(*Diagnostic)

// Diagnostic is a sdk.Hook that attaches runtime / env diagnostic attrs
// to every outbound signal. Existing attr keys are never overwritten.
type Diagnostic struct {
	version        string
	gitSHA         string
	envKeys        []string
	includeEnviron bool
	includeSys     bool
	maxEnviron     int
	service        string // optional mirror into attrs["service"] when missing

	once            sync.Once
	static          map[string]any
	startupEnviron  map[string]any // frozen at Agent/enricher init (integration startup)
}

// NewDiagnostic builds the default diagnostic enricher hook.
// By default it attaches:
//   - attrs["startup_environ"]: Startup env (frozen at Agent init)
//   - attrs["system_environ"]: System env (OS/shell + /etc/environment)
//   - attrs["process_environ"] (+ "environ"): Process env at send time
//   - attrs["sys"]: process / memory / disk / network snapshot
func NewDiagnostic(opts ...Option) *Diagnostic {
	d := &Diagnostic{
		envKeys:        append([]string(nil), DefaultEnvKeys...),
		includeEnviron: true,
		includeSys:     true,
		maxEnviron:     0, // unlimited after secret filter
	}
	for _, opt := range opts {
		opt(d)
	}
	if d.version == "" {
		d.version = LookupVersionFromEnv()
	}
	if d.gitSHA == "" {
		d.gitSHA = LookupGitSHAFromEnv()
	}
	// Capture startup env immediately when the enricher is created (Agent init).
	d.ensureStatic()
	return d
}

// WithVersion sets attrs["version"] (overrides VERSION / APP_VERSION env).
func WithVersion(v string) Option {
	return func(d *Diagnostic) { d.version = v }
}

// WithGitSHA sets attrs["git_sha"] (overrides GIT_SHA / GIT_COMMIT / COMMIT_SHA env).
func WithGitSHA(sha string) Option {
	return func(d *Diagnostic) { d.gitSHA = sha }
}

// WithEnvKeys replaces the allow-list used for the nested attrs["env"] map.
// Secret-looking keys are still skipped. Pass nil/empty to use DefaultEnvKeys.
func WithEnvKeys(keys ...string) Option {
	return func(d *Diagnostic) {
		if len(keys) == 0 {
			d.envKeys = append([]string(nil), DefaultEnvKeys...)
			return
		}
		d.envKeys = append([]string(nil), keys...)
	}
}

// WithIncludeEnviron includes environment snapshots:
//   - attrs["startup_environ"]: frozen at enricher/Agent init (Startup env)
//   - attrs["system_environ"]: OS/shell system env (System env)
//   - attrs["process_environ"] / attrs["environ"]: current process env (Process env)
// Secrets matching IsSecretEnvKey are omitted. Default is on.
func WithIncludeEnviron(enable bool) Option {
	return func(d *Diagnostic) { d.includeEnviron = enable }
}

// WithIncludeSys includes a process/disk/network snapshot under attrs["sys"].
// Default is on.
func WithIncludeSys(enable bool) Option {
	return func(d *Diagnostic) { d.includeSys = enable }
}

// WithMaxEnviron caps the number of keys in attrs["environ"] when enabled.
// Zero or negative means unlimited (still secret-filtered). Default 0 (unlimited).
func WithMaxEnviron(n int) Option {
	return func(d *Diagnostic) { d.maxEnviron = n }
}

// WithService mirrors the agent service name into attrs when the key is absent.
func WithService(service string) Option {
	return func(d *Diagnostic) { d.service = service }
}

func (d *Diagnostic) Name() string { return "diagnostic" }

func (d *Diagnostic) BeforeSend(sig signal.Signal) (signal.Signal, bool) {
	if d == nil || sig == nil {
		return sig, true
	}
	ev, ok := sig.(*signal.Event)
	if !ok || ev == nil {
		return sig, true
	}
	out := ev.Clone().(*signal.Event)
	if out.AttrsValue == nil {
		out.AttrsValue = make(map[string]any, 16)
	}
	for k, v := range d.buildAttrs() {
		if _, exists := out.AttrsValue[k]; exists {
			continue
		}
		out.AttrsValue[k] = v
	}
	return out, true
}

func (d *Diagnostic) ensureStatic() {
	d.once.Do(func() {
		host, _ := os.Hostname()
		cwd, _ := os.Getwd()
		d.static = map[string]any{
			"hostname":   host,
			"pid":        os.Getpid(),
			"go_version": runtime.Version(),
			"goos":       runtime.GOOS,
			"goarch":     runtime.GOARCH,
			"cwd":        cwd,
		}
		// Integration-side startup env (captured once when Agent/enricher starts).
		d.startupEnviron = CollectFilteredEnviron(d.maxEnviron)
	})
}

func (d *Diagnostic) buildAttrs() map[string]any {
	d.ensureStatic()

	attrs := make(map[string]any, len(d.static)+8)
	for k, v := range d.static {
		attrs[k] = v
	}
	if d.service != "" {
		attrs["service"] = d.service
	}
	if d.version != "" {
		attrs["version"] = d.version
	}
	if d.gitSHA != "" {
		attrs["git_sha"] = d.gitSHA
	}
	if _, profile := LookupProfile(); profile != "" {
		attrs["profile"] = profile
	}
	if env := CollectAllowListedEnv(d.envKeys); len(env) > 0 {
		attrs["env"] = env
	}
	if d.includeEnviron {
		if len(d.startupEnviron) > 0 {
			attrs["startup_environ"] = cloneMap(d.startupEnviron)
		}
		if sysEnv := CollectSystemEnviron(); len(sysEnv) > 0 {
			attrs["system_environ"] = sysEnv
		}
		if procEnv := CollectProcessEnviron(d.maxEnviron); len(procEnv) > 0 {
			attrs["process_environ"] = procEnv
			attrs["environ"] = procEnv // backward-compatible alias
		}
	}
	if d.includeSys {
		if sys := CollectSystemStatus(); len(sys) > 0 {
			attrs["sys"] = sys
		}
	}
	return attrs
}

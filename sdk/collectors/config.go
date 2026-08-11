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
	"os"
	"runtime"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/enricher"
)

type Config struct {
	Version string
	EnvKeys []string
}

// NewConfig builds a periodic config snapshot collector.
// When envKeys is empty, enricher.DefaultEnvKeys is used.
// Secret-looking keys are always skipped (see enricher.IsSecretEnvKey).
func NewConfig(version string, envKeys ...string) sdk.Collector {
	return &Config{Version: version, EnvKeys: envKeys}
}

func (c *Config) Name() string { return "config" }

func (c *Config) Collect(context.Context) ([]signal.Signal, error) {
	host, _ := os.Hostname()
	keys := c.EnvKeys
	if len(keys) == 0 {
		keys = enricher.DefaultEnvKeys
	}
	env := enricher.CollectAllowListedEnv(keys)
	attrs := map[string]any{
		"version":    c.Version,
		"go_version": runtime.Version(),
		"pid":        os.Getpid(),
		"env":        env,
	}
	if _, profile := enricher.LookupProfile(); profile != "" {
		attrs["profile"] = profile
	}
	if sha := enricher.LookupGitSHAFromEnv(); sha != "" {
		attrs["git_sha"] = sha
	}
	return []signal.Signal{&signal.Event{
		KindValue:    signal.KindConfig,
		TimeValue:    time.Now(),
		LevelValue:   "info",
		HostValue:    host,
		MessageValue: "config_snapshot",
		AttrsValue:   attrs,
	}}, nil
}

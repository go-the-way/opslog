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
	"log/slog"
	"os"
	"sync"
)

// stdLogger writes agent lifecycle / send diagnostics to stderr at Info+.
// It intentionally does not use slog.Default(), so host apps cannot silence it
// via slog.SetDefault / level overrides. It also must not go through Agent.Logger
// (that path enqueues OpsLog signals and can recurse).
var (
	stdLoggerMu sync.Mutex
	stdLogger   *slog.Logger
)

func defaultAgentLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func agentLog() *slog.Logger {
	stdLoggerMu.Lock()
	defer stdLoggerMu.Unlock()
	if stdLogger == nil {
		stdLogger = defaultAgentLogger()
	}
	return stdLogger
}

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

// Package tlog provides a stderr logger for client transports that is
// independent of slog.Default() (host apps often replace/silence that).
package tlog

import (
	"log/slog"
	"os"
	"sync"
)

var (
	mu sync.Mutex
	lg *slog.Logger
)

func defaultLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

// L returns a process-wide stderr Info+ logger for transport diagnostics.
func L() *slog.Logger {
	mu.Lock()
	defer mu.Unlock()
	if lg == nil {
		lg = defaultLogger()
	}
	return lg
}

// Set replaces the logger (tests only). Pass nil to restore the default.
func Set(l *slog.Logger) {
	mu.Lock()
	defer mu.Unlock()
	lg = l
}

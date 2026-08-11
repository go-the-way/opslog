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
	"sync/atomic"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/sdk"
)

type loggerImpl struct {
	agent  *agentImpl
	attrs  []signal.Attr
	closed *atomic.Bool
}

func (l *loggerImpl) Debug(msg string, attrs ...signal.Attr) { l.log("debug", msg, attrs...) }
func (l *loggerImpl) Info(msg string, attrs ...signal.Attr)  { l.log("info", msg, attrs...) }
func (l *loggerImpl) Warn(msg string, attrs ...signal.Attr)  { l.log("warn", msg, attrs...) }
func (l *loggerImpl) Error(msg string, attrs ...signal.Attr) { l.log("error", msg, attrs...) }

func (l *loggerImpl) With(attrs ...signal.Attr) sdk.Logger {
	merged := append(append([]signal.Attr{}, l.attrs...), attrs...)
	return &loggerImpl{agent: l.agent, attrs: merged, closed: l.closed}
}

func (l *loggerImpl) Sync() error { return l.agent.Sync() }

func (l *loggerImpl) log(level, msg string, attrs ...signal.Attr) {
	if l.closed.Load() {
		return
	}
	if !levelEnabled(l.agent.minLevel, level) {
		return
	}
	all := append(append([]signal.Attr{}, l.attrs...), attrs...)
	ev := &signal.Event{
		KindValue:    signal.KindLog,
		TimeValue:    time.Now(),
		LevelValue:   level,
		ServiceValue: l.agent.service,
		HostValue:    l.agent.host,
		MessageValue: msg,
		AttrsValue:   signal.AttrsToMap(all...),
	}
	if tid, ok := ev.AttrsValue["trace_id"].(string); ok {
		ev.TraceIDValue = tid
	}
	l.agent.enqueue(ev)
}

func levelEnabled(min, level string) bool {
	order := map[string]int{"debug": 10, "info": 20, "warn": 30, "error": 40, "fatal": 50, "panic": 60}
	return order[strings.ToLower(level)] >= order[strings.ToLower(min)]
}

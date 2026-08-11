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

package opslog4proc

import (
	"strings"
	"testing"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

type captureLogger struct {
	msgs  []string
	attrs [][]signal.Attr
}

func (l *captureLogger) Debug(string, ...signal.Attr) {}
func (l *captureLogger) Info(string, ...signal.Attr)  {}
func (l *captureLogger) Warn(string, ...signal.Attr)  {}
func (l *captureLogger) Error(msg string, attrs ...signal.Attr) {
	l.msgs = append(l.msgs, msg)
	l.attrs = append(l.attrs, attrs)
}
func (l *captureLogger) With(attrs ...signal.Attr) sdk.Logger { return l }
func (l *captureLogger) Sync() error                         { return nil }

func TestRecoverLogsStack(t *testing.T) {
	log := &captureLogger{}
	func() {
		defer Recover(log)
		panic("test-panic")
	}()
	if len(log.msgs) != 1 {
		t.Fatalf("expected 1 error log, got %d", len(log.msgs))
	}
	if !strings.Contains(log.msgs[0], "test-panic") {
		t.Fatalf("message missing panic value: %q", log.msgs[0])
	}
	m := signal.AttrsToMap(log.attrs[0]...)
	if m["panic"] != "test-panic" {
		t.Fatalf("panic attr = %v", m["panic"])
	}
	stack, _ := m["stack"].(string)
	if !strings.Contains(stack, "TestRecoverLogsStack") {
		t.Fatalf("stack missing test frame: %q", stack)
	}
}

func TestRecoverContinuePanic(t *testing.T) {
	log := &captureLogger{}
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected continue panic")
		}
	}()
	func() {
		defer Recover(log, panicopt.WithContinuePanic(true))
		panic("again")
	}()
}

func TestGuard(t *testing.T) {
	log := &captureLogger{}
	func() {
		defer Guard(log)()
		panic("guard-panic")
	}()
	if len(log.msgs) != 1 || !strings.Contains(log.msgs[0], "guard-panic") {
		t.Fatalf("unexpected logs: %v", log.msgs)
	}
}

func TestDo(t *testing.T) {
	log := &captureLogger{}
	Do(log, func() { panic("do-panic") })
	if len(log.msgs) != 1 || !strings.Contains(log.msgs[0], "do-panic") {
		t.Fatalf("unexpected logs: %v", log.msgs)
	}
}

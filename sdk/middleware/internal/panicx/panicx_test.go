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

package panicx

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

func TestReportPanic(t *testing.T) {
	log := &captureLogger{}
	ReportPanic(log, "boom", panicopt.WithMessage("custom"), panicopt.WithAttrs(sdk.String("k", "v")))
	if len(log.msgs) != 1 || !strings.Contains(log.msgs[0], "custom") || !strings.Contains(log.msgs[0], "boom") {
		t.Fatalf("unexpected message: %v", log.msgs)
	}
	m := signal.AttrsToMap(log.attrs[0]...)
	if m["panic"] != "boom" || m["k"] != "v" {
		t.Fatalf("attrs = %#v", m)
	}
	stack, _ := m["stack"].(string)
	if !strings.Contains(stack, "TestReportPanic") {
		t.Fatalf("stack missing frame: %q", stack)
	}
}

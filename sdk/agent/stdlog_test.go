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
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAgentStartLogsToStderrLogger(t *testing.T) {
	var agentBuf bytes.Buffer
	stdLoggerMu.Lock()
	stdLogger = slog.New(slog.NewTextHandler(&agentBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	stdLoggerMu.Unlock()
	t.Cleanup(func() {
		stdLoggerMu.Lock()
		stdLogger = nil
		stdLoggerMu.Unlock()
	})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	a, err := NewAgent(WithService("log-test"), WithEndpoint(srv.URL+"/ingest"), WithoutDiagnosticEnricher())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	if err := a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}

	agentOut := agentBuf.String()
	for _, want := range []string{"agent started", "agent transport ready", "service=log-test"} {
		if !strings.Contains(agentOut, want) {
			t.Fatalf("missing %q in agent logs:\n%s", want, agentOut)
		}
	}
}

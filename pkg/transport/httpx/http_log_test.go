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

package httpx

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-the-way/opslog/pkg/transport/internal/tlog"
)

func TestHTTPTransportConnectionLogs(t *testing.T) {
	var buf bytes.Buffer
	tlog.Set(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})))
	t.Cleanup(func() { tlog.Set(nil) })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if string(body) == "fail" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	tr, err := New("http", map[string]any{"url": srv.URL + "/ingest"})
	if err != nil {
		t.Fatal(err)
	}
	ht := tr.(*Transport)
	if err := ht.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{"http transport connecting", "http transport connected"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in logs:\n%s", want, out)
		}
	}

	buf.Reset()
	if err := ht.Send(context.Background(), []byte(`{"ok":true}`)); err != nil {
		t.Fatalf("expected first send ok, got %v", err)
	}
	if !strings.Contains(buf.String(), "http transport send ok") {
		t.Fatalf("missing send ok log:\n%s", buf.String())
	}

	buf.Reset()
	if err := ht.Send(context.Background(), []byte("fail")); err == nil {
		t.Fatal("expected non-2xx error")
	}
	if !strings.Contains(buf.String(), "http transport send failed") || !strings.Contains(buf.String(), "status=401") {
		t.Fatalf("missing send failed warn:\n%s", buf.String())
	}
}

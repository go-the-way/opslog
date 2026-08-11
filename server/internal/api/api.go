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

package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/go-the-way/opslog/pkg/query"
	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
	"github.com/go-the-way/opslog/server/internal/live"
)

type Server struct {
	Queryable contract.Queryable
	Archiver  contract.Archiver
	Restorer  contract.Restorer
	Hub       *live.Hub
}

func (s *Server) Mount(mux *http.ServeMux) {
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/signals", s.handleQuery)
	mux.HandleFunc("/api/metrics", s.handleMetrics)
	mux.HandleFunc("/api/probes", s.handleProbes)
	mux.HandleFunc("/api/configs", s.handleConfigs)
	mux.HandleFunc("/api/archives", s.handleArchives)
	mux.HandleFunc("/api/archives/restore", s.handleRestore)
	mux.HandleFunc("/api/tail", s.handleTail)
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	s.queryKind(w, r, signal.Kind(r.URL.Query().Get("kind")))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	s.queryKind(w, r, signal.KindMetric)
}

func (s *Server) handleProbes(w http.ResponseWriter, r *http.Request) {
	s.queryKind(w, r, signal.KindProbe)
}

func (s *Server) handleConfigs(w http.ResponseWriter, r *http.Request) {
	s.queryKind(w, r, signal.KindConfig)
}

func (s *Server) queryKind(w http.ResponseWriter, r *http.Request, kind signal.Kind) {
	if s.Queryable == nil {
		http.Error(w, "queryable output not configured", http.StatusServiceUnavailable)
		return
	}
	q := parseQuery(r)
	if kind != "" {
		q.Kind = kind
	}
	if q.Kind == "" {
		q.Kind = signal.KindLog
	}
	page, err := s.Queryable.Query(r.Context(), q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, pageToDTO(page))
}

func (s *Server) handleArchives(w http.ResponseWriter, r *http.Request) {
	if s.Archiver == nil {
		writeJSON(w, []any{})
		return
	}
	list, err := s.Archiver.ListArchives(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, list)
}

func (s *Server) handleRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.Restorer == nil {
		http.Error(w, "restore not supported", http.StatusServiceUnavailable)
		return
	}
	var body struct {
		ID        string `json:"id"`
		Overwrite bool   `json:"overwrite"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" {
		http.Error(w, "id required", http.StatusBadRequest)
		return
	}
	if err := s.Restorer.Restore(r.Context(), body.ID, query.RestoreOptions{ToHot: true, Overwrite: body.Overwrite}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleTail(w http.ResponseWriter, r *http.Request) {
	if s.Hub == nil {
		http.Error(w, "live hub unavailable", http.StatusServiceUnavailable)
		return
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer conn.Close(websocket.StatusNormalClosure, "")
	filter := query.Filter{
		Kind:     signal.Kind(r.URL.Query().Get("kind")),
		Keyword:  r.URL.Query().Get("keyword"),
		Levels:   splitCSV(r.URL.Query().Get("level")),
		Services: splitCSV(r.URL.Query().Get("service")),
		Hosts:    splitCSV(r.URL.Query().Get("host")),
	}
	ch, cancel, err := s.Hub.Subscribe(r.Context(), filter)
	if err != nil {
		return
	}
	defer cancel()
	for sig := range ch {
		b, _ := json.Marshal(eventDTO(sig))
		if err := conn.Write(r.Context(), websocket.MessageText, b); err != nil {
			return
		}
	}
}

func parseQuery(r *http.Request) query.Query {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	var from, to time.Time
	if v := q.Get("from"); v != "" {
		from, _ = time.Parse(time.RFC3339, v)
	}
	if v := q.Get("to"); v != "" {
		to, _ = time.Parse(time.RFC3339, v)
	}
	return query.Query{
		From: from, To: to,
		Levels: splitCSV(q.Get("level")),
		Services: splitCSV(q.Get("service")),
		Hosts:    splitCSV(q.Get("host")),
		TraceID:  q.Get("trace_id"),
		Keyword:  q.Get("keyword"),
		Limit:    limit,
		Offset:   offset,
	}
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func pageToDTO(p query.Page) map[string]any {
	items := make([]map[string]any, 0, len(p.Items))
	for _, it := range p.Items {
		items = append(items, eventDTO(it))
	}
	return map[string]any{"total": p.Total, "has_more": p.HasMore, "items": items}
}

func eventDTO(sig signal.Signal) map[string]any {
	return map[string]any{
		"kind":     sig.Kind(),
		"ts":       sig.Time().Format(time.RFC3339Nano),
		"level":    sig.Level(),
		"service":  sig.Service(),
		"host":     sig.Host(),
		"msg":      sig.Message(),
		"trace_id": sig.TraceID(),
		"attrs":    sig.Attrs(),
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

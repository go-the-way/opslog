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

package web

import (
	"crypto/subtle"
	"net/http"
)

// BasicAuth protects the Web Console static UI only.
// Callers must wrap the UI handler alone — not the whole mux —
// so /api/*, /ingest, and /stream remain unaffected.
func BasicAuth(username, password string, next http.Handler) http.Handler {
	if password == "" {
		return next
	}
	wantUser := []byte(username)
	wantPass := []byte(password)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok ||
			subtle.ConstantTimeCompare([]byte(user), wantUser) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), wantPass) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="OpsLog Console"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

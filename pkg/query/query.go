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

package query

import (
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
)

// Query describes a retrieval request against a Queryable output.
type Query struct {
	Kind      signal.Kind
	From      time.Time
	To        time.Time
	Levels    []string
	Services  []string
	Hosts     []string
	TraceID   string
	Keyword   string
	AttrMatch map[string]any
	Offset    int
	Limit     int
}

// Filter is a lightweight subscription filter for live streams.
type Filter struct {
	Kind     signal.Kind
	Levels   []string
	Services []string
	Hosts    []string
	Keyword  string
}

// Page is a paginated query result.
type Page struct {
	Total   int64
	Items   []signal.Signal
	HasMore bool
}

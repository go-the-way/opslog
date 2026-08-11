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

package contract

import (
	"context"
	"time"

	"github.com/go-the-way/opslog/pkg/query"
	"github.com/go-the-way/opslog/pkg/signal"
)

// OutputType identifies a persisted/exported destination.
type OutputType string

const (
	OutputFilesystem    OutputType = "filesystem"
	OutputMySQL         OutputType = "mysql"
	OutputClickHouse    OutputType = "clickhouse"
	OutputKafka         OutputType = "kafka"         // reserved
	OutputElasticsearch OutputType = "elasticsearch" // reserved
)

// Output is a server-side destination for processed signals.
// Priority implementations: filesystem / mysql / clickhouse.
type Output interface {
	Name() string
	Type() OutputType
	Write(ctx context.Context, batch []signal.Signal) error
	Flush(ctx context.Context) error
	Close(ctx context.Context) error
}

// OutputFactory builds an Output from opaque config.
type OutputFactory func(name string, cfg map[string]any) (Output, error)

// Queryable is an optional capability for outputs that support search.
type Queryable interface {
	Query(ctx context.Context, q query.Query) (query.Page, error)
}

// Archiver is an optional capability (filesystem) for hot -> cold compaction.
type Archiver interface {
	Archive(ctx context.Context, before time.Time) error
	ListArchives(ctx context.Context) ([]query.ArchiveInfo, error)
}

// Restorer is an optional capability (filesystem) for cold -> hot/query restore.
type Restorer interface {
	Restore(ctx context.Context, archiveID string, opts query.RestoreOptions) error
}

// LiveSubscriber is an optional capability for realtime fan-out (e.g. websocket UI).
type LiveSubscriber interface {
	Subscribe(ctx context.Context, filter query.Filter) (ch <-chan signal.Signal, cancel func(), err error)
}

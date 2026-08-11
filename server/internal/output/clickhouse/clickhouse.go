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

package clickhouse

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/query"
	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
)

type Output struct {
	name string
	db   *sql.DB
}

func New(name string, cfg map[string]any) (contract.Output, error) {
	dsn := cfgutil.String(cfg, "dsn", "")
	if dsn == "" {
		return nil, fmt.Errorf("clickhouse output: dsn is required")
	}
	if name == "" {
		name = "clickhouse"
	}
	db, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return nil, err
	}
	o := &Output{name: name, db: db}
	if err := o.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return o, nil
}

func (o *Output) Name() string              { return o.name }
func (o *Output) Type() contract.OutputType { return contract.OutputClickHouse }

func (o *Output) migrate(ctx context.Context) error {
	_, err := o.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS signals (
  ts DateTime64(3),
  kind LowCardinality(String),
  level LowCardinality(String),
  service LowCardinality(String),
  host String,
  message String,
  trace_id String,
  attrs String
) ENGINE = MergeTree
PARTITION BY toYYYYMMDD(ts)
ORDER BY (service, level, ts)
`)
	return err
}

func (o *Output) Write(ctx context.Context, batch []signal.Signal) error {
	if len(batch) == 0 {
		return nil
	}
	tx, err := o.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `INSERT INTO signals (ts, kind, level, service, host, message, trace_id, attrs)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, sig := range batch {
		attrs, _ := json.Marshal(sig.Attrs())
		ts := sig.Time()
		if ts.IsZero() {
			ts = time.Now()
		}
		if _, err := stmt.ExecContext(ctx, ts.UTC(), string(sig.Kind()), sig.Level(), sig.Service(), sig.Host(), sig.Message(), sig.TraceID(), string(attrs)); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

func (o *Output) Flush(context.Context) error { return nil }
func (o *Output) Close(context.Context) error  { return o.db.Close() }

func (o *Output) Query(ctx context.Context, q query.Query) (query.Page, error) {
	where := []string{"1=1"}
	args := []any{}
	if q.Kind != "" {
		where = append(where, "kind=?")
		args = append(args, string(q.Kind))
	}
	if !q.From.IsZero() {
		where = append(where, "ts>=?")
		args = append(args, q.From.UTC())
	}
	if !q.To.IsZero() {
		where = append(where, "ts<=?")
		args = append(args, q.To.UTC())
	}
	if len(q.Services) == 1 {
		where = append(where, "service=?")
		args = append(args, q.Services[0])
	}
	if q.Keyword != "" {
		where = append(where, "positionCaseInsensitive(message, ?) > 0")
		args = append(args, q.Keyword)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	var total uint64
	countSQL := "SELECT count() FROM signals WHERE " + strings.Join(where, " AND ")
	if err := o.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return query.Page{}, err
	}
	sqlStr := "SELECT ts, kind, level, service, host, message, trace_id, attrs FROM signals WHERE " +
		strings.Join(where, " AND ") + " ORDER BY ts DESC LIMIT ? OFFSET ?"
	args2 := append(append([]any{}, args...), limit, q.Offset)
	rows, err := o.db.QueryContext(ctx, sqlStr, args2...)
	if err != nil {
		return query.Page{}, err
	}
	defer rows.Close()
	var items []signal.Signal
	for rows.Next() {
		var (
			ts                                           time.Time
			kind, level, service, host, message, traceID string
			attrsRaw                                     string
		)
		if err := rows.Scan(&ts, &kind, &level, &service, &host, &message, &traceID, &attrsRaw); err != nil {
			return query.Page{}, err
		}
		var attrs map[string]any
		_ = json.Unmarshal([]byte(attrsRaw), &attrs)
		items = append(items, &signal.Event{
			KindValue: signal.Kind(kind), TimeValue: ts, LevelValue: level,
			ServiceValue: service, HostValue: host, MessageValue: message,
			TraceIDValue: traceID, AttrsValue: attrs,
		})
	}
	return query.Page{Total: int64(total), Items: items, HasMore: int64(q.Offset+len(items)) < int64(total)}, nil
}

var (
	_ contract.Output    = (*Output)(nil)
	_ contract.Queryable = (*Output)(nil)
)

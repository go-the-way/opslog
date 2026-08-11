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

package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

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
		return nil, fmt.Errorf("mysql output: dsn is required")
	}
	if name == "" {
		name = "mysql"
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)
	o := &Output{name: name, db: db}
	if err := o.migrate(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return o, nil
}

func (o *Output) Name() string              { return o.name }
func (o *Output) Type() contract.OutputType { return contract.OutputMySQL }

func (o *Output) migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS signals (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			kind VARCHAR(16) NOT NULL,
			ts DATETIME(3) NOT NULL,
			level VARCHAR(16),
			service VARCHAR(128),
			host VARCHAR(128),
			message TEXT,
			trace_id VARCHAR(64),
			attrs JSON,
			INDEX idx_ts (ts),
			INDEX idx_kind_ts (kind, ts),
			INDEX idx_service_ts (service, ts),
			INDEX idx_level_ts (level, ts),
			INDEX idx_trace (trace_id)
		) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
	}
	for _, s := range stmts {
		if _, err := o.db.ExecContext(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

func (o *Output) Write(ctx context.Context, batch []signal.Signal) error {
	if len(batch) == 0 {
		return nil
	}
	var b strings.Builder
	b.WriteString(`INSERT INTO signals (kind, ts, level, service, host, message, trace_id, attrs) VALUES `)
	args := make([]any, 0, len(batch)*8)
	for i, sig := range batch {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`(?,?,?,?,?,?,?,?)`)
		attrs, _ := json.Marshal(sig.Attrs())
		ts := sig.Time()
		if ts.IsZero() {
			ts = time.Now()
		}
		args = append(args, string(sig.Kind()), ts.UTC(), sig.Level(), sig.Service(), sig.Host(), sig.Message(), sig.TraceID(), string(attrs))
	}
	_, err := o.db.ExecContext(ctx, b.String(), args...)
	return err
}

func (o *Output) Flush(context.Context) error { return nil }

func (o *Output) Close(context.Context) error { return o.db.Close() }

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
	if len(q.Levels) > 0 {
		where = append(where, "level IN ("+placeholders(len(q.Levels))+")")
		for _, l := range q.Levels {
			args = append(args, l)
		}
	}
	if len(q.Services) == 1 {
		where = append(where, "service=?")
		args = append(args, q.Services[0])
	}
	if q.Keyword != "" {
		where = append(where, "message LIKE ?")
		args = append(args, "%"+q.Keyword+"%")
	}
	if q.TraceID != "" {
		where = append(where, "trace_id=?")
		args = append(args, q.TraceID)
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	countSQL := "SELECT COUNT(*) FROM signals WHERE " + strings.Join(where, " AND ")
	var total int64
	if err := o.db.QueryRowContext(ctx, countSQL, args...).Scan(&total); err != nil {
		return query.Page{}, err
	}
	sqlStr := "SELECT kind, ts, level, service, host, message, trace_id, attrs FROM signals WHERE " +
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
			kind, level, service, host, message, traceID string
			ts                                           time.Time
			attrsRaw                                     sql.NullString
		)
		if err := rows.Scan(&kind, &ts, &level, &service, &host, &message, &traceID, &attrsRaw); err != nil {
			return query.Page{}, err
		}
		var attrs map[string]any
		if attrsRaw.Valid {
			_ = json.Unmarshal([]byte(attrsRaw.String), &attrs)
		}
		items = append(items, &signal.Event{
			KindValue: signal.Kind(kind), TimeValue: ts, LevelValue: level,
			ServiceValue: service, HostValue: host, MessageValue: message,
			TraceIDValue: traceID, AttrsValue: attrs,
		})
	}
	return query.Page{Total: total, Items: items, HasMore: int64(q.Offset+len(items)) < total}, nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

var (
	_ contract.Output    = (*Output)(nil)
	_ contract.Queryable = (*Output)(nil)
)

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

package filesystem

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"github.com/go-the-way/opslog/pkg/codec"
	"github.com/go-the-way/opslog/pkg/query"
	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
)

type Output struct {
	name              string
	root              string
	archiveAfterDays  int
	archiveRetainDays int
	mu                sync.Mutex
	files             map[string]*os.File
}

func New(name string, cfg map[string]any) (contract.Output, error) {
	root := cfgutil.String(cfg, "path", "./data")
	if name == "" {
		name = "filesystem"
	}
	o := &Output{
		name:              name,
		root:              root,
		archiveAfterDays:  cfgutil.Int(cfg, "archive_after_days", 7),
		archiveRetainDays: cfgutil.Int(cfg, "archive_retain_days", 90),
		files:             make(map[string]*os.File),
	}
	for _, k := range []string{"log", "metric", "config", "probe"} {
		if err := os.MkdirAll(filepath.Join(root, "hot", k), 0o755); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "archive"), 0o755); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *Output) Name() string                 { return o.name }
func (o *Output) Type() contract.OutputType    { return contract.OutputFilesystem }

func (o *Output) Write(_ context.Context, batch []signal.Signal) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, sig := range batch {
		kind := string(sig.Kind())
		if kind == "" {
			kind = "log"
		}
		day := sig.Time().Format("2006-01-02")
		if sig.Time().IsZero() {
			day = time.Now().Format("2006-01-02")
		}
		path := filepath.Join(o.root, "hot", kind, day+".ndjson")
		f, err := o.openLocked(path)
		if err != nil {
			return err
		}
		b, err := codec.EncodeJSON(sig)
		if err != nil {
			return err
		}
		if _, err := f.Write(append(b, '\n')); err != nil {
			return err
		}
	}
	return nil
}

func (o *Output) openLocked(path string) (*os.File, error) {
	if f, ok := o.files[path]; ok {
		return f, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	o.files[path] = f
	return f, nil
}

func (o *Output) Flush(context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	for _, f := range o.files {
		_ = f.Sync()
	}
	return nil
}

func (o *Output) Close(context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	for p, f := range o.files {
		_ = f.Close()
		delete(o.files, p)
	}
	return nil
}

func (o *Output) Query(_ context.Context, q query.Query) (query.Page, error) {
	kind := string(q.Kind)
	if kind == "" {
		kind = "log"
	}
	dir := filepath.Join(o.root, "hot", kind)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return query.Page{}, nil
		}
		return query.Page{}, err
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".ndjson") {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	sort.Sort(sort.Reverse(sort.StringSlice(paths)))
	var items []signal.Signal
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
		for sc.Scan() {
			ev, err := codec.DecodeJSON(sc.Bytes())
			if err != nil {
				continue
			}
			if matchQuery(q, ev) {
				items = append(items, ev)
			}
		}
		_ = f.Close()
	}
	// newest first
	sort.Slice(items, func(i, j int) bool { return items[i].Time().After(items[j].Time()) })
	total := int64(len(items))
	if q.Offset > 0 && q.Offset < len(items) {
		items = items[q.Offset:]
	} else if q.Offset >= len(items) {
		items = nil
	}
	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}
	return query.Page{Total: total, Items: items, HasMore: hasMore}, nil
}

func matchQuery(q query.Query, ev *signal.Event) bool {
	if !q.From.IsZero() && ev.Time().Before(q.From) {
		return false
	}
	if !q.To.IsZero() && ev.Time().After(q.To) {
		return false
	}
	if len(q.Levels) > 0 && !in(q.Levels, ev.Level()) {
		return false
	}
	if len(q.Services) > 0 && !in(q.Services, ev.Service()) {
		return false
	}
	if len(q.Hosts) > 0 && !in(q.Hosts, ev.Host()) {
		return false
	}
	if q.TraceID != "" && ev.TraceID() != q.TraceID {
		return false
	}
	if q.Keyword != "" && !strings.Contains(strings.ToLower(ev.Message()), strings.ToLower(q.Keyword)) {
		return false
	}
	return true
}

func in(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

func (o *Output) Archive(ctx context.Context, before time.Time) error {
	_ = ctx
	o.mu.Lock()
	for p, f := range o.files {
		_ = f.Close()
		delete(o.files, p)
	}
	o.mu.Unlock()

	for _, kind := range []string{"log", "metric", "config", "probe"} {
		dir := filepath.Join(o.root, "hot", kind)
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".ndjson") {
				continue
			}
			day := strings.TrimSuffix(e.Name(), ".ndjson")
			t, err := time.ParseInLocation("2006-01-02", day, time.Local)
			if err != nil || !t.Before(before) {
				continue
			}
			src := filepath.Join(dir, e.Name())
			id := fmt.Sprintf("%s-%s", kind, day)
			dst := filepath.Join(o.root, "archive", id+".ndjson.gz")
			metaPath := filepath.Join(o.root, "archive", id+".meta.json")
			if err := gzipFile(src, dst); err != nil {
				return err
			}
			info, _ := os.Stat(dst)
			sum, _ := fileSHA(dst)
			meta := query.ArchiveInfo{
				ID: id, Path: dst, Kind: kind,
				From: t, To: t.Add(24*time.Hour - time.Nanosecond),
				SizeBytes: info.Size(), Checksum: sum, CreatedAt: time.Now(),
			}
			b, _ := json.MarshalIndent(meta, "", "  ")
			_ = os.WriteFile(metaPath, b, 0o644)
			_ = os.Remove(src)
		}
	}
	if o.archiveRetainDays > 0 {
		cutoff := time.Now().AddDate(0, 0, -o.archiveRetainDays)
		_ = o.pruneArchives(cutoff)
	}
	return nil
}

func (o *Output) ListArchives(context.Context) ([]query.ArchiveInfo, error) {
	dir := filepath.Join(o.root, "archive")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []query.ArchiveInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var info query.ArchiveInfo
		if json.Unmarshal(b, &info) == nil {
			out = append(out, info)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].From.After(out[j].From) })
	return out, nil
}

func (o *Output) Restore(_ context.Context, archiveID string, opts query.RestoreOptions) error {
	metaPath := filepath.Join(o.root, "archive", archiveID+".meta.json")
	gzPath := filepath.Join(o.root, "archive", archiveID+".ndjson.gz")
	b, err := os.ReadFile(metaPath)
	if err != nil {
		return err
	}
	var info query.ArchiveInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return err
	}
	kind := info.Kind
	if kind == "" {
		kind = "log"
	}
	day := info.From.Format("2006-01-02")
	dst := filepath.Join(o.root, "hot", kind, day+".ndjson")
	if !opts.ToHot && opts.ReadOnly {
		// read-only mount: extract beside archive for query path reuse
		dst = filepath.Join(o.root, "archive", "restored", kind, day+".ndjson")
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(dst); err == nil && !opts.Overwrite {
		return fmt.Errorf("restore target exists: %s", dst)
	}
	return gunzipFile(gzPath, dst)
}

func (o *Output) pruneArchives(before time.Time) error {
	list, err := o.ListArchives(context.Background())
	if err != nil {
		return err
	}
	for _, a := range list {
		if a.From.Before(before) {
			_ = os.Remove(a.Path)
			_ = os.Remove(filepath.Join(o.root, "archive", a.ID+".meta.json"))
		}
	}
	return nil
}

func gzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	zw := gzip.NewWriter(out)
	if _, err := io.Copy(zw, in); err != nil {
		_ = zw.Close()
		return err
	}
	return zw.Close()
}

func gunzipFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	zr, err := gzip.NewReader(in)
	if err != nil {
		return err
	}
	defer zr.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, zr)
	return err
}

func fileSHA(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

var (
	_ contract.Output    = (*Output)(nil)
	_ contract.Queryable = (*Output)(nil)
	_ contract.Archiver  = (*Output)(nil)
	_ contract.Restorer  = (*Output)(nil)
)

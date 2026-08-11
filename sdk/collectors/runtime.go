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

package collectors

import (
	"context"
	"runtime"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/sdk"
)

type Runtime struct{}

func NewRuntime() sdk.Collector { return Runtime{} }

func (Runtime) Name() string { return "runtime" }

func (Runtime) Collect(context.Context) ([]signal.Signal, error) {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	now := time.Now()
	attrs := map[string]any{
		"goroutines": runtime.NumGoroutine(),
		"heap_alloc": ms.HeapAlloc,
		"heap_sys":   ms.HeapSys,
		"gc_pause_ns": ms.PauseNs[(ms.NumGC+255)%256],
		"num_gc":     ms.NumGC,
	}
	return []signal.Signal{&signal.Event{
		KindValue:    signal.KindMetric,
		TimeValue:    now,
		LevelValue:   "info",
		MessageValue: "runtime",
		AttrsValue:   attrs,
	}}, nil
}

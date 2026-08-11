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

package live

import (
	"context"
	"strings"
	"sync"

	"github.com/go-the-way/opslog/pkg/query"
	"github.com/go-the-way/opslog/pkg/signal"
)

// Hub fans out live signals to console subscribers.
type Hub struct {
	mu   sync.RWMutex
	subs map[int]chan signal.Signal
	next int
}

func NewHub() *Hub {
	return &Hub{subs: make(map[int]chan signal.Signal)}
}

func (h *Hub) Publish(sig signal.Signal) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, ch := range h.subs {
		select {
		case ch <- sig:
		default:
		}
	}
}

func (h *Hub) Subscribe(ctx context.Context, filter query.Filter) (<-chan signal.Signal, func(), error) {
	ch := make(chan signal.Signal, 128)
	h.mu.Lock()
	id := h.next
	h.next++
	h.subs[id] = ch
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		if c, ok := h.subs[id]; ok {
			delete(h.subs, id)
			close(c)
		}
		h.mu.Unlock()
	}

	out := make(chan signal.Signal, 128)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				cancel()
				return
			case sig, ok := <-ch:
				if !ok {
					return
				}
				if match(filter, sig) {
					select {
					case out <- sig:
					default:
					}
				}
			}
		}
	}()
	return out, cancel, nil
}

func match(f query.Filter, sig signal.Signal) bool {
	if f.Kind != "" && sig.Kind() != f.Kind {
		return false
	}
	if len(f.Levels) > 0 && !contains(f.Levels, sig.Level()) {
		return false
	}
	if len(f.Services) > 0 && !contains(f.Services, sig.Service()) {
		return false
	}
	if len(f.Hosts) > 0 && !contains(f.Hosts, sig.Host()) {
		return false
	}
	if f.Keyword != "" && !strings.Contains(strings.ToLower(sig.Message()), strings.ToLower(f.Keyword)) {
		return false
	}
	return true
}

func contains(ss []string, v string) bool {
	for _, s := range ss {
		if s == v {
			return true
		}
	}
	return false
}

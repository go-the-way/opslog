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

package policy

import (
	"strings"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/pkg/transport"
)

// LevelPolicy routes error+ (and optionally config) to a reliable transport.
type LevelPolicy struct {
	DefaultName string
	ErrorName   string
	ConfigName  string
}

func NewLevelPolicy(defaultName, errorName string) *LevelPolicy {
	return &LevelPolicy{DefaultName: defaultName, ErrorName: errorName, ConfigName: errorName}
}

func (p *LevelPolicy) Name() string { return "level" }

func (p *LevelPolicy) Select(sig signal.Signal, available []transport.Transport) []transport.Transport {
	if len(available) == 0 {
		return nil
	}
	byName := map[string]transport.Transport{}
	for _, t := range available {
		byName[t.Name()] = t
		byName[string(t.Type())] = t
	}
	want := p.DefaultName
	level := strings.ToLower(sig.Level())
	if sig.Kind() == signal.KindConfig && p.ConfigName != "" {
		want = p.ConfigName
	} else if level == "error" || level == "fatal" || level == "panic" {
		if p.ErrorName != "" {
			want = p.ErrorName
		}
	}
	if want != "" {
		if t, ok := byName[want]; ok {
			return []transport.Transport{t}
		}
	}
	return []transport.Transport{available[0]}
}

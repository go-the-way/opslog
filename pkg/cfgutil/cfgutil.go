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

package cfgutil

import (
	"fmt"
	"strconv"
	"time"
)

func String(cfg map[string]any, key, def string) string {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case string:
		if t == "" {
			return def
		}
		return t
	default:
		return fmt.Sprint(t)
	}
}

func Int(cfg map[string]any, key string, def int) int {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	case string:
		n, err := strconv.Atoi(t)
		if err != nil {
			return def
		}
		return n
	default:
		return def
	}
}

func Bool(cfg map[string]any, key string, def bool) bool {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		b, err := strconv.ParseBool(t)
		if err != nil {
			return def
		}
		return b
	default:
		return def
	}
}

func Duration(cfg map[string]any, key string, def time.Duration) time.Duration {
	if cfg == nil {
		return def
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return def
	}
	switch t := v.(type) {
	case time.Duration:
		return t
	case string:
		d, err := time.ParseDuration(t)
		if err != nil {
			return def
		}
		return d
	case int:
		return time.Duration(t) * time.Second
	case int64:
		return time.Duration(t) * time.Second
	case float64:
		return time.Duration(t) * time.Second
	default:
		return def
	}
}

func StringSlice(cfg map[string]any, key string) []string {
	if cfg == nil {
		return nil
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			out = append(out, fmt.Sprint(item))
		}
		return out
	default:
		return nil
	}
}

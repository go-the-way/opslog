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

package signal

// Attr is a typed key/value pair attached to a signal.
type Attr struct {
	Key   string
	Value any
}

func String(key, value string) Attr  { return Attr{Key: key, Value: value} }
func Int(key string, value int) Attr { return Attr{Key: key, Value: value} }
func Int64(key string, value int64) Attr {
	return Attr{Key: key, Value: value}
}
func Float64(key string, value float64) Attr {
	return Attr{Key: key, Value: value}
}
func Bool(key string, value bool) Attr { return Attr{Key: key, Value: value} }
func Any(key string, value any) Attr   { return Attr{Key: key, Value: value} }

// AttrsToMap converts attrs into a map. Later keys override earlier ones.
func AttrsToMap(attrs ...Attr) map[string]any {
	if len(attrs) == 0 {
		return nil
	}
	m := make(map[string]any, len(attrs))
	for _, a := range attrs {
		if a.Key == "" {
			continue
		}
		m[a.Key] = a.Value
	}
	return m
}

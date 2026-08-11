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

import "time"

// Event is the default Signal implementation used on the wire and in pipelines.
type Event struct {
	KindValue    Kind           `json:"kind"`
	TimeValue    time.Time      `json:"ts"`
	LevelValue   string         `json:"level,omitempty"`
	ServiceValue string         `json:"service,omitempty"`
	HostValue    string         `json:"host,omitempty"`
	MessageValue string         `json:"msg,omitempty"`
	TraceIDValue string         `json:"trace_id,omitempty"`
	AttrsValue   map[string]any `json:"attrs,omitempty"`
	RawValue     []byte         `json:"-"`
}

func (e *Event) Kind() Kind            { return e.KindValue }
func (e *Event) Time() time.Time       { return e.TimeValue }
func (e *Event) Level() string         { return e.LevelValue }
func (e *Event) Service() string       { return e.ServiceValue }
func (e *Event) Host() string          { return e.HostValue }
func (e *Event) Message() string       { return e.MessageValue }
func (e *Event) TraceID() string       { return e.TraceIDValue }
func (e *Event) Attrs() map[string]any { return e.AttrsValue }
func (e *Event) Raw() []byte           { return e.RawValue }

// Clone returns a shallow copy with attrs map duplicated.
func (e *Event) Clone() Signal {
	if e == nil {
		return nil
	}
	cp := *e
	if e.AttrsValue != nil {
		cp.AttrsValue = make(map[string]any, len(e.AttrsValue))
		for k, v := range e.AttrsValue {
			cp.AttrsValue[k] = v
		}
	}
	if e.RawValue != nil {
		cp.RawValue = append([]byte(nil), e.RawValue...)
	}
	return &cp
}

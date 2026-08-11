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

package codec

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
)

// EncodeJSON serializes a Signal to JSON bytes.
func EncodeJSON(sig signal.Signal) ([]byte, error) {
	if sig == nil {
		return nil, fmt.Errorf("codec: nil signal")
	}
	ev, ok := sig.(*signal.Event)
	if !ok {
		ev = &signal.Event{
			KindValue:    sig.Kind(),
			TimeValue:    sig.Time(),
			LevelValue:   sig.Level(),
			ServiceValue: sig.Service(),
			HostValue:    sig.Host(),
			MessageValue: sig.Message(),
			TraceIDValue: sig.TraceID(),
			AttrsValue:   sig.Attrs(),
		}
	}
	return json.Marshal(ev)
}

// DecodeJSON parses JSON into an Event.
func DecodeJSON(payload []byte) (*signal.Event, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, err
	}
	ev := &signal.Event{RawValue: append([]byte(nil), payload...)}

	if v, ok := raw["kind"]; ok {
		var k string
		_ = json.Unmarshal(v, &k)
		ev.KindValue = signal.Kind(k)
	}
	if ev.KindValue == "" {
		ev.KindValue = signal.KindLog
	}
	if v, ok := raw["ts"]; ok {
		var ts string
		if err := json.Unmarshal(v, &ts); err == nil && ts != "" {
			if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
				ev.TimeValue = t
			} else if t, err := time.Parse(time.RFC3339, ts); err == nil {
				ev.TimeValue = t
			}
		}
	}
	if v, ok := raw["level"]; ok {
		_ = json.Unmarshal(v, &ev.LevelValue)
	}
	if v, ok := raw["service"]; ok {
		_ = json.Unmarshal(v, &ev.ServiceValue)
	}
	if v, ok := raw["host"]; ok {
		_ = json.Unmarshal(v, &ev.HostValue)
	}
	if v, ok := raw["msg"]; ok {
		_ = json.Unmarshal(v, &ev.MessageValue)
	} else if v, ok := raw["message"]; ok {
		_ = json.Unmarshal(v, &ev.MessageValue)
	}
	if v, ok := raw["trace_id"]; ok {
		_ = json.Unmarshal(v, &ev.TraceIDValue)
	}
	if v, ok := raw["attrs"]; ok {
		_ = json.Unmarshal(v, &ev.AttrsValue)
	}
	return ev, nil
}

// DecodePlain builds a log Event from a single text line.
func DecodePlain(payload []byte) *signal.Event {
	msg := strings.TrimSpace(string(payload))
	return &signal.Event{
		KindValue:    signal.KindLog,
		TimeValue:    time.Now(),
		LevelValue:   "info",
		MessageValue: msg,
		RawValue:     append([]byte(nil), payload...),
	}
}

// EncodeBatchJSON encodes multiple signals as {"events":[...]} .
func EncodeBatchJSON(batch []signal.Signal) ([]byte, error) {
	events := make([]*signal.Event, 0, len(batch))
	for _, s := range batch {
		b, err := EncodeJSON(s)
		if err != nil {
			return nil, err
		}
		var ev signal.Event
		if err := json.Unmarshal(b, &ev); err != nil {
			return nil, err
		}
		events = append(events, &ev)
	}
	return json.Marshal(map[string]any{"events": events})
}

// DecodeBatchJSON accepts a single event or {"events":[...]} / raw array.
func DecodeBatchJSON(payload []byte) ([]*signal.Event, error) {
	trim := strings.TrimSpace(string(payload))
	if trim == "" {
		return nil, fmt.Errorf("codec: empty payload")
	}
	if strings.HasPrefix(trim, "[") {
		var arr []json.RawMessage
		if err := json.Unmarshal(payload, &arr); err != nil {
			return nil, err
		}
		out := make([]*signal.Event, 0, len(arr))
		for _, item := range arr {
			ev, err := DecodeJSON(item)
			if err != nil {
				return nil, err
			}
			out = append(out, ev)
		}
		return out, nil
	}
	var envelope struct {
		Events []json.RawMessage `json:"events"`
	}
	if err := json.Unmarshal(payload, &envelope); err == nil && len(envelope.Events) > 0 {
		out := make([]*signal.Event, 0, len(envelope.Events))
		for _, item := range envelope.Events {
			ev, err := DecodeJSON(item)
			if err != nil {
				return nil, err
			}
			out = append(out, ev)
		}
		return out, nil
	}
	ev, err := DecodeJSON(payload)
	if err != nil {
		return nil, err
	}
	return []*signal.Event{ev}, nil
}

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

package decoder

import (
	"strings"
	"time"

	"github.com/go-the-way/opslog/pkg/codec"
	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
)

type JSON struct{ name string }

func NewJSON(name string, _ map[string]any) (contract.Decoder, error) {
	if name == "" {
		name = "json"
	}
	return &JSON{name: name}, nil
}

func (d *JSON) Name() string { return d.name }

func (d *JSON) Decode(remoteAddr string, payload []byte) (signal.Signal, error) {
	ev, err := codec.DecodeJSON(payload)
	if err != nil {
		return nil, err
	}
	if ev.HostValue == "" && remoteAddr != "" {
		ev.HostValue = stripPort(remoteAddr)
	}
	if ev.TimeValue.IsZero() {
		ev.TimeValue = time.Now()
	}
	return ev, nil
}

type Plain struct{ name string }

func NewPlain(name string, _ map[string]any) (contract.Decoder, error) {
	if name == "" {
		name = "plain"
	}
	return &Plain{name: name}, nil
}

func (d *Plain) Name() string { return d.name }

func (d *Plain) Decode(remoteAddr string, payload []byte) (signal.Signal, error) {
	ev := codec.DecodePlain(payload)
	if remoteAddr != "" {
		ev.HostValue = stripPort(remoteAddr)
	}
	return ev, nil
}

func stripPort(addr string) string {
	if i := strings.LastIndex(addr, ":"); i > 0 {
		// keep IPv6 brackets simple: if more than one ':', leave as-is unless ] present
		if strings.Count(addr, ":") == 1 {
			return addr[:i]
		}
		if br := strings.Index(addr, "]:"); br >= 0 {
			return addr[:br+1]
		}
	}
	return addr
}

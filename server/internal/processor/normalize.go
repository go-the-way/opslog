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

package processor

import (
	"context"
	"strings"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
)

// Normalize fills default fields on *signal.Event.
type Normalize struct {
	name string
}

func NewNormalize(name string, _ map[string]any) (contract.Processor, error) {
	if name == "" {
		name = "normalize"
	}
	return &Normalize{name: name}, nil
}

func (n *Normalize) Name() string { return n.name }

func (n *Normalize) Process(_ context.Context, sig signal.Signal) (signal.Signal, bool, error) {
	ev, ok := sig.(*signal.Event)
	if !ok {
		return sig, true, nil
	}
	if ev.KindValue == "" {
		ev.KindValue = signal.KindLog
	}
	if ev.TimeValue.IsZero() {
		ev.TimeValue = time.Now()
	}
	if ev.LevelValue == "" {
		ev.LevelValue = "info"
	} else {
		ev.LevelValue = strings.ToLower(ev.LevelValue)
	}
	return ev, true, nil
}

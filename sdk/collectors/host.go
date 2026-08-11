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
	"os"
	"time"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/enricher"
)

type Host struct{}

func NewHost() sdk.Collector { return Host{} }

func (Host) Name() string { return "host" }

func (Host) Collect(context.Context) ([]signal.Signal, error) {
	host, _ := os.Hostname()
	now := time.Now()
	attrs := enricher.CollectSystemStatus()
	if attrs == nil {
		attrs = map[string]any{}
	}
	attrs["hostname"] = host
	return []signal.Signal{&signal.Event{
		KindValue:    signal.KindMetric,
		TimeValue:    now,
		LevelValue:   "info",
		HostValue:    host,
		MessageValue: "host_resources",
		AttrsValue:   attrs,
	}}, nil
}

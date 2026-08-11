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

package sdk

import (
	"context"

	"github.com/go-the-way/opslog/pkg/signal"
)

// Collector periodically (or on demand) produces diagnostic signals:
// runtime metrics, host resources/config, connectivity probes, etc.
type Collector interface {
	Name() string
	Collect(ctx context.Context) ([]signal.Signal, error)
}

// Probe checks one dependency/connectivity target.
type Probe interface {
	Name() string
	Target() string
	Check(ctx context.Context) (signal.Signal, error)
}

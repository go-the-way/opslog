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

package contract

import (
	"context"

	"github.com/go-the-way/opslog/pkg/signal"
)

// Processor transforms or filters a signal inside the pipeline.
// keep=false means the signal should be dropped.
type Processor interface {
	Name() string
	Process(ctx context.Context, sig signal.Signal) (out signal.Signal, keep bool, err error)
}

// ProcessorFactory builds a Processor from opaque config.
type ProcessorFactory func(name string, cfg map[string]any) (Processor, error)

// Pipeline runs an ordered processor chain.
type Pipeline interface {
	Name() string
	Process(ctx context.Context, sig signal.Signal) (out signal.Signal, keep bool, err error)
	Processors() []Processor
}

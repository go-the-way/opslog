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

package pipeline

import (
	"context"

	"github.com/go-the-way/opslog/pkg/signal"
	"github.com/go-the-way/opslog/server/contract"
)

type Chain struct {
	name       string
	processors []contract.Processor
}

func New(name string, processors ...contract.Processor) *Chain {
	if name == "" {
		name = "default"
	}
	return &Chain{name: name, processors: processors}
}

func (c *Chain) Name() string { return c.name }

func (c *Chain) Processors() []contract.Processor { return append([]contract.Processor(nil), c.processors...) }

func (c *Chain) Process(ctx context.Context, sig signal.Signal) (signal.Signal, bool, error) {
	cur := sig
	for _, p := range c.processors {
		out, keep, err := p.Process(ctx, cur)
		if err != nil {
			return cur, false, err
		}
		if !keep {
			return out, false, nil
		}
		cur = out
	}
	return cur, true, nil
}

var _ contract.Pipeline = (*Chain)(nil)

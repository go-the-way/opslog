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

// Dispatcher accepts decoded signals from Inputs and feeds the processing pipeline.
type Dispatcher interface {
	Dispatch(ctx context.Context, sig signal.Signal) error
}

// BatchDispatcher is an optional capability for inputs that already batch.
type BatchDispatcher interface {
	DispatchBatch(ctx context.Context, batch []signal.Signal) error
}

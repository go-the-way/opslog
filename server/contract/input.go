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

import "context"

// Input is a server-side ingress adapter (udp/tcp/http/websocket/grpc listener).
type Input interface {
	Name() string
	Type() string
	Start(ctx context.Context, dispatcher Dispatcher) error
	Stop(ctx context.Context) error
}

// InputFactory builds an Input from opaque config.
type InputFactory func(name string, cfg map[string]any) (Input, error)

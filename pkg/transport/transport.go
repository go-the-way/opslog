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

package transport

import "context"

// Transport is the client-side send abstraction used by the SDK.
// Implementations: udp / tcp / http / websocket / grpc.
type Transport interface {
	Name() string
	Type() Type
	Start(ctx context.Context) error
	Send(ctx context.Context, payload []byte) error
	SendBatch(ctx context.Context, payloads [][]byte) error
	Flush(ctx context.Context) error
	Close() error
	Healthy() bool
}

// Factory builds a Transport from opaque config.
type Factory func(name string, cfg map[string]any) (Transport, error)

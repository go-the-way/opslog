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

package signal

import "time"

// Signal is the unified contract shared by SDK and Server.
// Concrete payloads should implement this interface so pipelines stay decoupled.
type Signal interface {
	Kind() Kind
	Time() time.Time
	Level() string
	Service() string
	Host() string
	Message() string
	TraceID() string
	Attrs() map[string]any
	Raw() []byte
}

// Cloneable is an optional capability for processors that need mutation safety.
type Cloneable interface {
	Clone() Signal
}

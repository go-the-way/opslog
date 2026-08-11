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

// Package sdk defines embeddable client contracts for OpsLog (interfaces and Attr helpers).
//
// Implementations live in subpackages:
//
//	sdk/agent                  — NewAgent, options, runtime
//	sdk/formatter              — JSON formatter
//	sdk/policy                 — level-based transport policy
//	sdk/transport              — transport constructors (UDP/TCP/HTTP/WS/gRPC)
//	sdk/collectors             — runtime / host / config / probe collectors
//	sdk/enricher               — default diagnostic context hook (env / runtime)
//	sdk/middleware/panicopt    — shared panic options
//	sdk/middleware/opslog4gin  — Gin recovery middleware
//	sdk/middleware/opslog4proc — Recover / Guard / Do / ReportPanic (process-level)
//	sdk/integration            — env bootstrap + GinRecovery/Guard/Main helpers
//
// Typical composition:
//
//	Agent
//	  ├─ Logger
//	  ├─ Collector(s)
//	  ├─ Hook(s)
//	  ├─ Formatter
//	  ├─ Policy
//	  └─ Transport(s)
//
// Signal kinds align with package signal: log / metric / config / probe.
// See docs/sdk.md for embedding guidance.
package sdk

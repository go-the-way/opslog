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

// Package enricher attaches diagnostic runtime/env/system context to outbound signals.
//
// By default NewDiagnostic attaches:
//   - profile / version / host / go runtime
//   - attrs["startup_environ"]: Startup env (Agent init, frozen)
//   - attrs["system_environ"]: System env (OS/shell + /etc/environment)
//   - attrs["process_environ"]: Process env at send time (also as "environ")
//   - attrs["sys"]: process, memory, disk, network snapshot
//
// Agent enables NewDiagnostic() by default; disable with agent.WithoutDiagnosticEnricher().
package enricher

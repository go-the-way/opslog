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

// Package integration provides a minimal embed helper for OpsLog Agent:
// env-based bootstrap, Close, Gin recovery, and process Guard.
//
// Typical usage:
//
//	integration.MustInitFromEnv(integration.WithService("my-api"), integration.WithVersion("1.0.0"))
//	defer integration.Close()
//	r.Use(integration.GinRecovery())
//	defer integration.Guard()()
//
// When OPSLOG_ENABLE is not "T"/"true"/"1", Init is a no-op and
// GinRecovery/Guard fall back to safe defaults (gin.Recovery / empty func).
package integration

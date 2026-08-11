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

// Package opslog4proc reports Go panics through an OpsLog Logger.
//
// Import path: github.com/go-the-way/opslog/sdk/middleware/opslog4proc
//
// Thin public wrapper around the shared panicx core. Process-level helpers
// (defer Recover / Guard / Do / ReportPanic) — not HTTP middleware; use
// sdk/middleware/opslog4gin for Gin.
//
// Shared options live in sdk/middleware/panicopt (WithContinuePanic / WithAttrs / WithMessage).
package opslog4proc

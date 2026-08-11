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

// Package middleware groups framework-specific OpsLog helpers.
//
// Import a concrete subpackage when needed:
//
//	sdk/middleware/panicopt    — shared panic options (WithContinuePanic / WithAttrs / WithMessage)
//	sdk/middleware/opslog4gin  — Gin recovery middleware
//	sdk/middleware/opslog4proc — process-level Recover / Guard / Do / ReportPanic
//
// Shared panic reporting implementation lives in sdk/middleware/internal/panicx
// (not a public API). The core sdk package does not depend on web frameworks.
package middleware

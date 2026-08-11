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

// Package opslog4gin provides Gin middleware helpers for OpsLog SDK.
//
// Import path: github.com/go-the-way/opslog/sdk/middleware/opslog4gin
//
// Recovery reports panics via the shared panicx core (same logging path as
// opslog4proc), then writes an HTTP 500 response. Panic-reporting options come
// from sdk/middleware/panicopt; gin-specific response shape uses Response /
// RecoveryResponse. Import this subpackage only when you use Gin.
package opslog4gin

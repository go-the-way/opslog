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

// Package panicopt holds shared panic-reporting options for OpsLog middleware.
//
// Import path: github.com/go-the-way/opslog/sdk/middleware/panicopt
//
// Use with opslog4proc and opslog4gin:
//
//	defer opslog4proc.Guard(logger, panicopt.WithAttrs(...))()
//	r.Use(opslog4gin.Recovery(logger, panicopt.WithContinuePanic(true)))
package panicopt

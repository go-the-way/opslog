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

package opslog4proc

import (
	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/middleware/internal/panicx"
	"github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

// Recover recovers a panic in the current goroutine, logs it with a runtime
// call stack via logger, and does not continue the panic unless
// panicopt.WithContinuePanic(true) is set.
//
// Must be used with defer:
//
//	defer opslog4proc.Recover(logger)
//	defer opslog4proc.Recover(logger, panicopt.WithContinuePanic(true))
//
// Prefer Guard when you want a closure form: defer opslog4proc.Guard(logger)()
func Recover(logger sdk.Logger, opts ...panicopt.Option) {
	panicx.Recover(logger, opts...)
}

// RecoverAgent is like Recover but accepts an Agent and uses its Logger.
//
//	defer opslog4proc.RecoverAgent(agent)
func RecoverAgent(agent sdk.Agent, opts ...panicopt.Option) {
	panicx.RecoverAgent(agent, opts...)
}

// Guard returns a function suitable for defer that recovers and reports panics.
//
//	defer opslog4proc.Guard(logger)()
//	defer opslog4proc.Guard(logger, panicopt.WithAttrs(sdk.String("component", "worker")))()
func Guard(logger sdk.Logger, opts ...panicopt.Option) func() {
	return panicx.Guard(logger, opts...)
}

// GuardAgent is like Guard but accepts an Agent and uses its Logger.
//
//	defer opslog4proc.GuardAgent(agent)()
func GuardAgent(agent sdk.Agent, opts ...panicopt.Option) func() {
	return panicx.GuardAgent(agent, opts...)
}

// Do runs fn and recovers panics via Recover (logs stack; does not continue the panic unless opted in).
//
//	opslog4proc.Do(logger, func() {
//	    // work that may panic
//	})
func Do(logger sdk.Logger, fn func(), opts ...panicopt.Option) {
	panicx.Do(logger, fn, opts...)
}

// DoAgent is like Do but accepts an Agent and uses its Logger.
//
//	opslog4proc.DoAgent(agent, func() { /* ... */ })
func DoAgent(agent sdk.Agent, fn func(), opts ...panicopt.Option) {
	panicx.DoAgent(agent, fn, opts...)
}

// ReportPanic logs a recovered panic value with the Go call stack.
// It does not call recover(); use it from custom middleware after you recover.
// By default it does not continue the panic; pass panicopt.WithContinuePanic(true) to panic again.
func ReportPanic(logger sdk.Logger, recovered any, opts ...panicopt.Option) {
	panicx.ReportPanic(logger, recovered, opts...)
}

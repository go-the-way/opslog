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

package panicx

import (
	"fmt"
	"runtime/debug"

	"github.com/go-the-way/opslog/sdk"
	"github.com/go-the-way/opslog/sdk/middleware/panicopt"
)

// Stack returns the current goroutine's call stack.
func Stack() string { return string(debug.Stack()) }

// Recover recovers a panic in the current goroutine and reports it.
// Must be used with defer.
func Recover(logger sdk.Logger, opts ...panicopt.Option) {
	r := recover()
	if r == nil {
		return
	}
	ReportPanic(logger, r, opts...)
}

// RecoverAgent is like Recover but accepts an Agent and uses its Logger.
func RecoverAgent(agent sdk.Agent, opts ...panicopt.Option) {
	r := recover()
	if r == nil {
		return
	}
	if agent == nil {
		if panicopt.Apply(opts).ContinuePanic {
			panic(r)
		}
		return
	}
	ReportPanic(agent.Logger(), r, opts...)
}

// ReportPanic logs a recovered panic value with the Go call stack.
// It does not call recover(); use it after you already recovered.
// By default it does not continue the panic; pass panicopt.WithContinuePanic(true) to panic again.
func ReportPanic(logger sdk.Logger, recovered any, opts ...panicopt.Option) {
	o := panicopt.Apply(opts)
	msg := o.Msg
	if msg == "" {
		msg = "panic recovered"
	}
	attrs := make([]sdk.Attr, 0, 3+len(o.Attrs))
	attrs = append(attrs,
		sdk.String("panic", fmt.Sprint(recovered)),
		sdk.String("stack", Stack()),
		sdk.Bool("recovered", true),
	)
	attrs = append(attrs, o.Attrs...)

	if logger != nil {
		logger.Error(fmt.Sprintf("%s: %v", msg, recovered), attrs...)
		_ = logger.Sync()
	}
	if o.ContinuePanic {
		panic(recovered)
	}
}

// Guard returns a function suitable for defer that recovers and reports panics.
// recover must run in the deferred closure itself (not via Recover), so that
// Go's recover rules apply when used as: defer Guard(logger)()
//
//	defer Guard(logger)()
//	defer Guard(logger, panicopt.WithAttrs(...))()
func Guard(logger sdk.Logger, opts ...panicopt.Option) func() {
	return func() {
		r := recover()
		if r == nil {
			return
		}
		ReportPanic(logger, r, opts...)
	}
}

// GuardAgent is like Guard but accepts an Agent and uses its Logger.
//
//	defer GuardAgent(agent)()
func GuardAgent(agent sdk.Agent, opts ...panicopt.Option) func() {
	return func() {
		r := recover()
		if r == nil {
			return
		}
		if agent == nil {
			if panicopt.Apply(opts).ContinuePanic {
				panic(r)
			}
			return
		}
		ReportPanic(agent.Logger(), r, opts...)
	}
}

// Do runs fn and recovers panics via Recover (logs stack; does not continue the panic unless opted in).
func Do(logger sdk.Logger, fn func(), opts ...panicopt.Option) {
	if fn == nil {
		return
	}
	defer Recover(logger, opts...)
	fn()
}

// DoAgent is like Do but accepts an Agent and uses its Logger.
func DoAgent(agent sdk.Agent, fn func(), opts ...panicopt.Option) {
	if fn == nil {
		return
	}
	defer RecoverAgent(agent, opts...)
	fn()
}

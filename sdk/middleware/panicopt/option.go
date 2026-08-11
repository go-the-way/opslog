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

package panicopt

import "github.com/go-the-way/opslog/sdk"

// Options configures panic recovery / reporting.
type Options struct {
	ContinuePanic bool
	Attrs         []sdk.Attr
	Msg           string
}

// Option mutates Options.
type Option func(*Options)

// WithContinuePanic controls whether Recover / ReportPanic panics again after logging.
// Default is false: recover and log, do not continue the panic.
func WithContinuePanic(continuePanic bool) Option {
	return func(o *Options) { o.ContinuePanic = continuePanic }
}

// WithAttrs attaches extra attributes to the panic error log.
func WithAttrs(attrs ...sdk.Attr) Option {
	return func(o *Options) { o.Attrs = append(o.Attrs, attrs...) }
}

// WithMessage overrides the default log message prefix ("panic recovered").
func WithMessage(msg string) Option {
	return func(o *Options) { o.Msg = msg }
}

// Apply returns Options after applying opts (nil-safe).
func Apply(opts []Option) *Options {
	o := &Options{}
	for _, opt := range opts {
		if opt != nil {
			opt(o)
		}
	}
	return o
}

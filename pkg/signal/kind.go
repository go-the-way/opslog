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

package signal

// Kind identifies the category of a collected signal.
type Kind string

const (
	// KindLog is an application or system log event.
	KindLog Kind = "log"
	// KindMetric is a numeric/resource measurement sample.
	KindMetric Kind = "metric"
	// KindConfig is a host/app configuration snapshot.
	KindConfig Kind = "config"
	// KindProbe is a connectivity/health probe result.
	KindProbe Kind = "probe"
)

// Valid reports whether k is a known kind.
func (k Kind) Valid() bool {
	switch k {
	case KindLog, KindMetric, KindConfig, KindProbe:
		return true
	default:
		return false
	}
}

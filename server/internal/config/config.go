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

package config

import (
	"fmt"
	"os"

	"github.com/go-the-way/opslog/pkg/cfgutil"
	"gopkg.in/yaml.v3"
)

type File struct {
	HTTP     HTTPConfig       `yaml:"http"`
	Web      WebConfig        `yaml:"web"`
	Inputs   []map[string]any `yaml:"inputs"`
	Outputs  []map[string]any `yaml:"outputs"`
	Pipeline []map[string]any `yaml:"pipeline"`
}

// HTTPConfig is the shared HTTP listen address (console + ingest routes).
// Token, when non-empty, requires Authorization: Bearer <token> on
// /ingest and /stream only. It does not protect the Web Console
// (use web.basic_auth for that).
type HTTPConfig struct {
	Listen string `yaml:"listen"`
	Token  string `yaml:"token"`
}

// WebConfig holds Web Console settings (static UI + query APIs).
type WebConfig struct {
	BasicAuth BasicAuthConfig `yaml:"basic_auth"`
}

// BasicAuthConfig is HTTP Basic Auth for the Web Console static UI only.
// Query APIs (/api/*) and ingest (/ingest, /stream, TCP/gRPC) are not covered.
//
// Policy:
//   - enabled=false → Basic Auth off
//   - enabled=true (default) with a non-empty password → Basic Auth on
//   - empty password → Basic Auth off (even when enabled=true)
// Defaults when the whole basic_auth block is omitted: enabled, opslog/opslog.
type BasicAuthConfig struct {
	Enabled  *bool  `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// Active reports whether Basic Auth should be enforced.
func (b BasicAuthConfig) Active() bool {
	if b.Enabled != nil && !*b.Enabled {
		return false
	}
	return b.Password != ""
}

type Component struct {
	Name string
	Type string
	Cfg  map[string]any
}

func Load(path string) (*File, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	if f.HTTP.Listen == "" {
		f.HTTP.Listen = ":8600"
	}
	applyWebBasicAuthDefaults(&f.Web.BasicAuth)
	if len(f.Outputs) == 0 {
		return nil, fmt.Errorf("config: at least one output is required")
	}
	return &f, nil
}

func applyWebBasicAuthDefaults(ba *BasicAuthConfig) {
	zero := ba.Enabled == nil && ba.Username == "" && ba.Password == ""
	if zero {
		on := true
		ba.Enabled = &on
		ba.Username = "opslog"
		ba.Password = "opslog"
		return
	}
	if ba.Enabled == nil {
		on := true
		ba.Enabled = &on
	}
	if ba.Username == "" {
		ba.Username = "opslog"
	}
}

func ParseComponent(m map[string]any) (Component, error) {
	name := cfgutil.String(m, "name", "")
	typ := cfgutil.String(m, "type", "")
	if typ == "" {
		return Component{}, fmt.Errorf("component type is required")
	}
	if name == "" {
		name = typ
	}
	cfg := make(map[string]any, len(m))
	for k, v := range m {
		if k == "name" || k == "type" {
			continue
		}
		cfg[k] = v
	}
	return Component{Name: name, Type: typ, Cfg: cfg}, nil
}

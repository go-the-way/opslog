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

package enricher

import (
	"os"
	"strings"
)

// ProfileEnvKeys are checked in order; the first set value becomes attrs["profile"].
var ProfileEnvKeys = []string{"APP_ENV", "ENV", "GO_ENV", "PROFILE"}

// DefaultEnvKeys is the allow-list used for the nested attrs["env"] map
// (and by ConfigCollector when no keys are provided).
var DefaultEnvKeys = []string{
	"APP_ENV", "ENV", "GO_ENV", "PROFILE",
	"APP_NAME", "SERVICE", "SERVICE_NAME",
	"VERSION", "APP_VERSION", "GIT_SHA", "GIT_COMMIT", "COMMIT_SHA",
	"HOSTNAME", "POD_NAME", "POD_NAMESPACE", "NAMESPACE",
	"REGION", "ZONE", "IDC", "DC",
}

// SystemEnvKeys are typical OS / shell environment variables shown under
// attrs["system_environ"] (System env).
var SystemEnvKeys = []string{
	"PATH", "HOME", "USER", "LOGNAME", "SHELL", "TERM", "TMPDIR", "TMP", "TEMP",
	"PWD", "OLDPWD", "LANG", "LANGUAGE", "LC_ALL", "LC_CTYPE", "LC_MESSAGES",
	"TZ", "HOSTNAME", "HOST", "HOSTTYPE", "OSTYPE", "MACHTYPE",
	"EDITOR", "VISUAL", "PAGER", "DISPLAY", "SSH_CLIENT", "SSH_CONNECTION",
	"SSH_TTY", "XDG_RUNTIME_DIR", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME",
	"GOOS", "GOARCH", "GOROOT", "GOPATH", "GOBIN", "GOTOOLCHAIN",
}

// secretSubstrings match case-insensitively as substrings of the env key name.
var secretSubstrings = []string{
	"PASSWORD", "PASSWD", "SECRET", "TOKEN", "PRIVATE",
	"CREDENTIAL", "APIKEY", "API_KEY", "ACCESS_KEY", "SECRET_KEY",
	"AUTH", "BEARER", "SESSION", "COOKIE",
}

// IsSecretEnvKey reports whether an environment variable name looks sensitive.
// Matching is substring-based on the uppercased key (e.g. DB_PASSWORD, MY_TOKEN).
// Bare suffix KEY is also treated as secret (AWS_ACCESS_KEY_ID), but not short
// keys that merely end with unrelated letters.
func IsSecretEnvKey(key string) bool {
	if key == "" {
		return false
	}
	u := strings.ToUpper(key)
	for _, s := range secretSubstrings {
		if strings.Contains(u, s) {
			return true
		}
	}
	// *KEY* but avoid over-matching tiny names; require KEY as a segment.
	if strings.Contains(u, "_KEY") || strings.HasSuffix(u, "KEY") || strings.HasPrefix(u, "KEY_") {
		return true
	}
	return false
}

// LookupProfile returns the first non-empty value among ProfileEnvKeys and the key used.
func LookupProfile() (key, value string) {
	for _, k := range ProfileEnvKeys {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			return k, v
		}
	}
	return "", ""
}

// CollectAllowListedEnv returns values for allow-listed keys that are set,
// skipping secret-looking names even if they appear in keys.
func CollectAllowListedEnv(keys []string) map[string]any {
	if len(keys) == 0 {
		keys = DefaultEnvKeys
	}
	out := make(map[string]any, len(keys))
	for _, k := range keys {
		if k == "" || IsSecretEnvKey(k) {
			continue
		}
		if v, ok := os.LookupEnv(k); ok {
			out[k] = v
		}
	}
	return out
}

// CollectFilteredEnviron returns a filtered snapshot of os.Environ().
// Secret-looking keys are omitted. If max > 0, at most max entries are kept
// (stable iteration order is not guaranteed beyond "first max after filter").
func CollectFilteredEnviron(max int) map[string]any {
	environ := os.Environ()
	out := make(map[string]any, len(environ))
	for _, e := range environ {
		k, v, ok := strings.Cut(e, "=")
		if !ok || k == "" || IsSecretEnvKey(k) {
			continue
		}
		out[k] = v
		if max > 0 && len(out) >= max {
			break
		}
	}
	return out
}

// IsSystemEnvKey reports whether key looks like an OS/shell system variable.
func IsSystemEnvKey(key string) bool {
	if key == "" {
		return false
	}
	u := strings.ToUpper(key)
	for _, k := range SystemEnvKeys {
		if u == k {
			return true
		}
	}
	if strings.HasPrefix(u, "LC_") || strings.HasPrefix(u, "XDG_") {
		return true
	}
	if strings.HasPrefix(u, "GO") && len(u) <= 16 { // GOOS, GOPATH, GOTOOLCHAIN…
		return true
	}
	return false
}

// CollectSystemEnviron returns OS/shell-oriented env vars from the current process
// (and merges /etc/environment on Unix when readable).
func CollectSystemEnviron() map[string]any {
	out := make(map[string]any, len(SystemEnvKeys)+8)
	for k, v := range readEtcEnvironment() {
		if k == "" || IsSecretEnvKey(k) {
			continue
		}
		out[k] = v
	}
	for _, k := range SystemEnvKeys {
		if IsSecretEnvKey(k) {
			continue
		}
		if v, ok := os.LookupEnv(k); ok {
			out[k] = v
		}
	}
	// Also pick up LC_* / XDG_* present in the process.
	for _, e := range os.Environ() {
		k, v, ok := strings.Cut(e, "=")
		if !ok || k == "" || IsSecretEnvKey(k) || !IsSystemEnvKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}

// CollectProcessEnviron is an alias of CollectFilteredEnviron for clarity:
// the full current process environment (secrets filtered).
func CollectProcessEnviron(max int) map[string]any {
	return CollectFilteredEnviron(max)
}

// LookupVersionFromEnv returns VERSION or APP_VERSION if set.
func LookupVersionFromEnv() string {
	for _, k := range []string{"VERSION", "APP_VERSION"} {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			return v
		}
	}
	return ""
}

// LookupGitSHAFromEnv returns GIT_SHA, GIT_COMMIT, or COMMIT_SHA if set.
func LookupGitSHAFromEnv() string {
	for _, k := range []string{"GIT_SHA", "GIT_COMMIT", "COMMIT_SHA"} {
		if v, ok := os.LookupEnv(k); ok && v != "" {
			return v
		}
	}
	return ""
}

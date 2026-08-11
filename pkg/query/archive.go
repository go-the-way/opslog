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

package query

import "time"

// ArchiveInfo describes one archived filesystem segment.
type ArchiveInfo struct {
	ID        string    `json:"id"`
	Path      string    `json:"path"`
	Kind      string    `json:"kind"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	Count     int64     `json:"count"`
	SizeBytes int64     `json:"size_bytes"`
	Checksum  string    `json:"checksum"`
	CreatedAt time.Time `json:"created_at"`
}

// RestoreOptions controls how an archive is restored.
type RestoreOptions struct {
	// ToHot moves data back into the hot searchable path.
	ToHot bool
	// ReadOnly mounts archive for query without mutating hot storage.
	ReadOnly bool
	// Overwrite replaces existing hot segments when colliding.
	Overwrite bool
}

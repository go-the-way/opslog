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

//go:build !unix

package enricher

import gopsutilnet "github.com/shirou/gopsutil/v3/net"

func diskUsage(string) (map[string]any, error) { return nil, errUnsupported("disk") }
func processRSS() (uint64, error)               { return 0, errUnsupported("rss") }
func processVMS() (uint64, error)               { return 0, errUnsupported("vms") }
func processNumFDs() (int, error)               { return -1, errUnsupported("fds") }

func networkStats() []map[string]any {
	counters, err := gopsutilnet.IOCounters(true)
	if err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(counters))
	for _, c := range counters {
		if c.Name == "lo" || (c.BytesRecv == 0 && c.BytesSent == 0) {
			continue
		}
		out = append(out, map[string]any{
			"name":       c.Name,
			"rx_bytes":   c.BytesRecv,
			"rx_human":   FormatBytes(c.BytesRecv),
			"tx_bytes":   c.BytesSent,
			"tx_human":   FormatBytes(c.BytesSent),
			"rx_packets": c.PacketsRecv,
			"tx_packets": c.PacketsSent,
			"note":       "cumulative counters since boot (not instantaneous rate)",
		})
	}
	return out
}

type unsupportedError string

func (e unsupportedError) Error() string { return string(e) + ": unsupported platform" }

func errUnsupported(name string) error { return unsupportedError(name) }

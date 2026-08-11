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

//go:build unix

package enricher

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"

	gopsutilnet "github.com/shirou/gopsutil/v3/net"
	"golang.org/x/sys/unix"
)

func diskUsage(path string) (map[string]any, error) {
	var st unix.Statfs_t
	if err := unix.Statfs(path, &st); err != nil {
		return nil, err
	}
	bsize := uint64(st.Bsize)
	if bsize == 0 {
		bsize = diskBlockSize(&st)
	}
	total := st.Blocks * bsize
	free := st.Bavail * bsize
	used := total - free
	var usedPct float64
	if total > 0 {
		usedPct = float64(used) * 100 / float64(total)
	}
	return map[string]any{
		"path":           path,
		"total":          total,
		"total_human":    FormatBytes(total),
		"free":           free,
		"free_human":     FormatBytes(free),
		"used":           used,
		"used_human":     FormatBytes(used),
		"used_percent":   fmt.Sprintf("%.1f", usedPct),
		"inodes_total":   st.Files,
		"inodes_free":    st.Ffree,
	}, nil
}

func processRSS() (uint64, error) {
	if runtime.GOOS == "linux" {
		return readLinuxStatus("VmRSS:")
	}
	// Darwin / other unix: best-effort via rusage (max RSS, not current).
	var ru unix.Rusage
	if err := unix.Getrusage(unix.RUSAGE_SELF, &ru); err != nil {
		return 0, err
	}
	// ru_maxrss is KB on Linux, bytes on Darwin — normalize roughly.
	rss := uint64(ru.Maxrss)
	if runtime.GOOS == "linux" {
		rss *= 1024
	}
	return rss, nil
}

func processVMS() (uint64, error) {
	if runtime.GOOS == "linux" {
		return readLinuxStatus("VmSize:")
	}
	return 0, fmt.Errorf("vms: unsupported")
}

func processNumFDs() (int, error) {
	if runtime.GOOS != "linux" {
		return -1, fmt.Errorf("fds: unsupported")
	}
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return -1, err
	}
	return len(entries), nil
}

func readLinuxStatus(key string) (uint64, error) {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if !strings.HasPrefix(line, key) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("parse %s", key)
		}
		kb, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0, err
		}
		return kb * 1024, nil
	}
	return 0, fmt.Errorf("%s not found", key)
}

func networkStats() []map[string]any {
	counters, err := gopsutilnet.IOCounters(true)
	if err != nil || len(counters) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(counters))
	for _, c := range counters {
		name := c.Name
		if name == "lo" || name == "lo0" || strings.HasPrefix(name, "awdl") || strings.HasPrefix(name, "llw") {
			continue
		}
		if c.BytesRecv == 0 && c.BytesSent == 0 {
			continue
		}
		out = append(out, map[string]any{
			"name":       name,
			"rx_bytes":   c.BytesRecv,
			"rx_human":   FormatBytes(c.BytesRecv),
			"tx_bytes":   c.BytesSent,
			"tx_human":   FormatBytes(c.BytesSent),
			"rx_packets": c.PacketsRecv,
			"tx_packets": c.PacketsSent,
			"rx_errs":    c.Errin,
			"tx_errs":    c.Errout,
			"note":       "cumulative counters since boot (not instantaneous rate)",
		})
	}
	return out
}

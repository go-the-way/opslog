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
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"time"
)

const sysCacheTTL = 2 * time.Second

var (
	sysMu    sync.Mutex
	sysCache map[string]any
	sysAt    time.Time
)

// CollectSystemStatus returns a snapshot of process / memory / disk / network.
// Results are cached briefly to keep per-log overhead low.
func CollectSystemStatus() map[string]any {
	sysMu.Lock()
	defer sysMu.Unlock()
	if sysCache != nil && time.Since(sysAt) < sysCacheTTL {
		return cloneMap(sysCache)
	}
	sysCache = buildSystemStatus()
	sysAt = time.Now()
	return cloneMap(sysCache)
}

func buildSystemStatus() map[string]any {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)

	cwd, _ := os.Getwd()
	out := map[string]any{
		"pid":             os.Getpid(),
		"ppid":            os.Getppid(),
		"uid":             os.Getuid(),
		"gid":             os.Getgid(),
		"num_cpu":         runtime.NumCPU(),
		"goroutines":      runtime.NumGoroutine(),
		"go_version":      runtime.Version(),
		"goos":            runtime.GOOS,
		"goarch":          runtime.GOARCH,
		"cwd":             cwd,
		"mem_alloc":       ms.Alloc,
		"mem_alloc_human": FormatBytes(ms.Alloc),
		"mem_sys":         ms.Sys,
		"mem_sys_human":   FormatBytes(ms.Sys),
		"mem_heap_inuse":  ms.HeapInuse,
		"mem_heap_sys":    ms.HeapSys,
		"gc_num":          ms.NumGC,
		"gc_pause_ns":     ms.PauseNs[(ms.NumGC+255)%256],
	}

	if rss, err := processRSS(); err == nil && rss > 0 {
		out["proc_rss"] = rss
		out["proc_rss_human"] = FormatBytes(rss)
	}
	if vms, err := processVMS(); err == nil && vms > 0 {
		out["proc_vms"] = vms
		out["proc_vms_human"] = FormatBytes(vms)
	}
	if nfd, err := processNumFDs(); err == nil && nfd >= 0 {
		out["proc_fds"] = nfd
	}

	for _, path := range []string{cwd, "/"} {
		if path == "" {
			continue
		}
		if d, err := diskUsage(path); err == nil && d != nil {
			out["disk"] = d
			break
		}
	}

	if nets := networkStats(); len(nets) > 0 {
		out["net"] = nets
	}
	if addrs := interfaceAddrs(); len(addrs) > 0 {
		out["net_addrs"] = addrs
	}
	return out
}

func interfaceAddrs() []map[string]any {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	out := make([]map[string]any, 0, len(ifaces))
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, _ := iface.Addrs()
		list := make([]string, 0, len(addrs))
		for _, a := range addrs {
			list = append(list, a.String())
		}
		if len(list) == 0 {
			continue
		}
		out = append(out, map[string]any{
			"name":  iface.Name,
			"mac":   iface.HardwareAddr.String(),
			"addrs": list,
		})
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// FormatBytes renders a human-readable size.
func FormatBytes(n uint64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
		tb = 1024 * gb
	)
	switch {
	case n >= tb:
		return fmt.Sprintf("%.2f TiB", float64(n)/float64(tb))
	case n >= gb:
		return fmt.Sprintf("%.2f GiB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.2f MiB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KiB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

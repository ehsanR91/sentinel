package monitoring

// procfs.go — direct /proc reads replacing gopsutil for system-level metrics.
// Each function is minimal: open → read → parse → close with no heap allocs
// beyond the returned values.

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ─── boot-time cache (shared with processes.go) ──────────────────────────────

// already declared in processes.go:
//   var procBootTimeCache cachedBootTime

// ─── /proc/loadavg ──────────────────────────────────────────────────────────

// readLoadAvg reads load averages from /proc/loadavg.
// Returns (load1, load5, load15, ok).
func readLoadAvg() (float64, float64, float64, bool) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0, false
	}
	// format: "0.00 0.01 0.05 1/123 456"
	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return 0, 0, 0, false
	}
	l1, e1 := strconv.ParseFloat(fields[0], 64)
	l5, e2 := strconv.ParseFloat(fields[1], 64)
	l15, e3 := strconv.ParseFloat(fields[2], 64)
	if e1 != nil || e2 != nil || e3 != nil {
		return 0, 0, 0, false
	}
	return l1, l5, l15, true
}

// ─── /proc/uptime ────────────────────────────────────────────────────────────

// readUptimeSeconds reads system uptime from /proc/uptime.
func readUptimeSeconds() (uint64, bool) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, false
	}
	idx := bytes.IndexByte(data, ' ')
	if idx < 0 {
		idx = len(data)
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(string(data[:idx])), 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return uint64(f), true
}

// ─── /proc/meminfo ───────────────────────────────────────────────────────────

type memStats struct {
	MemTotal     uint64
	MemAvailable uint64
	MemFree      uint64
	SwapTotal    uint64
	SwapFree     uint64
}

// readMemInfo parses /proc/meminfo in a single pass.
func readMemInfo() (memStats, bool) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return memStats{}, false
	}
	defer f.Close()

	var s memStats
	got := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && got < 5 {
		line := scanner.Text()
		var field *uint64
		switch {
		case strings.HasPrefix(line, "MemTotal:"):
			field = &s.MemTotal
		case strings.HasPrefix(line, "MemAvailable:"):
			field = &s.MemAvailable
		case strings.HasPrefix(line, "MemFree:"):
			field = &s.MemFree
		case strings.HasPrefix(line, "SwapTotal:"):
			field = &s.SwapTotal
		case strings.HasPrefix(line, "SwapFree:"):
			field = &s.SwapFree
		default:
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			v, err := strconv.ParseUint(parts[1], 10, 64)
			if err == nil {
				*field = v * 1024 // kB → bytes
				got++
			}
		}
	}
	return s, s.MemTotal > 0
}

// ─── /proc/stat (CPU) ────────────────────────────────────────────────────────

type cpuJiffies struct {
	user    uint64
	nice    uint64
	system  uint64
	idle    uint64
	iowait  uint64
	irq     uint64
	softirq uint64
	steal   uint64
}

func (j cpuJiffies) total() uint64 {
	return j.user + j.nice + j.system + j.idle + j.iowait + j.irq + j.softirq + j.steal
}
func (j cpuJiffies) busy() uint64 {
	return j.total() - j.idle - j.iowait
}

// cpuDeltaCache holds the previous CPU jiffy snapshot for delta calculation.
var cpuDeltaCache struct {
	mu      sync.Mutex
	perCore []cpuJiffies
	agg     cpuJiffies
}

// readCPUPercent reads /proc/stat and returns (perCore %, aggregate %, ok).
// Uses a cached previous sample for delta; safe to call from a single goroutine.
func readCPUPercent() (perCore []float64, aggregate float64, ok bool) {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var aggNow cpuJiffies
	var coresNow []cpuJiffies

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "cpu") {
			break
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		var j cpuJiffies
		vals := []*uint64{&j.user, &j.nice, &j.system, &j.idle, &j.iowait, &j.irq, &j.softirq, &j.steal}
		for i, p := range vals {
			if i+1 < len(fields) {
				v, _ := strconv.ParseUint(fields[i+1], 10, 64)
				*p = v
			}
		}
		if fields[0] == "cpu" {
			aggNow = j
		} else {
			coresNow = append(coresNow, j)
		}
	}

	if aggNow.total() == 0 {
		return
	}

	cpuDeltaCache.mu.Lock()
	defer cpuDeltaCache.mu.Unlock()

	prev := cpuDeltaCache.agg
	prevCores := cpuDeltaCache.perCore
	cpuDeltaCache.agg = aggNow
	cpuDeltaCache.perCore = coresNow

	// compute aggregate
	dtotal := aggNow.total() - prev.total()
	dbusy := int64(aggNow.busy()) - int64(prev.busy())
	if dtotal > 0 && prev.total() > 0 {
		aggregate = round2(float64(dbusy) / float64(dtotal) * 100)
		if aggregate < 0 {
			aggregate = 0
		}
	}

	// compute per-core
	perCore = make([]float64, len(coresNow))
	for i, c := range coresNow {
		if i < len(prevCores) {
			ct := c.total() - prevCores[i].total()
			cb := int64(c.busy()) - int64(prevCores[i].busy())
			if ct > 0 && prevCores[i].total() > 0 {
				v := round2(float64(cb) / float64(ct) * 100)
				if v < 0 {
					v = 0
				}
				perCore[i] = v
			}
		}
	}

	ok = true
	return
}

// ─── /proc/net/dev ───────────────────────────────────────────────────────────

type netDevStats struct {
	RxBytes uint64
	TxBytes uint64
}

// readNetDevAgg reads /proc/net/dev and sums all non-loopback interfaces.
func readNetDevAgg() (netDevStats, bool) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return netDevStats{}, false
	}
	defer f.Close()

	var total netDevStats
	scanner := bufio.NewScanner(f)
	scanner.Scan() // header 1
	scanner.Scan() // header 2
	for scanner.Scan() {
		line := scanner.Text()
		colonIdx := strings.IndexByte(line, ':')
		if colonIdx < 0 {
			continue
		}
		iface := strings.TrimSpace(line[:colonIdx])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(strings.TrimSpace(line[colonIdx+1:]))
		// fields: rx_bytes rx_packets rx_errs rx_drop rx_fifo rx_frame rx_compressed rx_multicast
		//         tx_bytes tx_packets ...
		if len(fields) < 9 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		total.RxBytes += rx
		total.TxBytes += tx
	}
	return total, true
}

// ─── /proc/stat btime (cached) ──────────────────────────────────────────────

// cachedBootTimeSeconds is already in processes.go; reused here.

// ─── /proc/sys/kernel/hostname ───────────────────────────────────────────────

var cachedHostname struct {
	sync.Once
	value string
}

func getHostname() string {
	cachedHostname.Do(func() {
		if data, err := os.ReadFile("/proc/sys/kernel/hostname"); err == nil {
			cachedHostname.value = strings.TrimSpace(string(data))
		} else {
			// fallback
			if h, err := os.Hostname(); err == nil {
				cachedHostname.value = h
			}
		}
	})
	return cachedHostname.value
}

// ─── /proc/version ───────────────────────────────────────────────────────────

var cachedKernelVersion struct {
	sync.Once
	value string
}

func getKernelVersion() string {
	cachedKernelVersion.Do(func() {
		if data, err := os.ReadFile("/proc/version"); err == nil {
			// "Linux version 6.8.0-111-generic ..."
			fields := strings.Fields(string(data))
			if len(fields) >= 3 {
				cachedKernelVersion.value = fields[2]
			}
		}
	})
	return cachedKernelVersion.value
}

// ─── /etc/os-release ─────────────────────────────────────────────────────────

var cachedOSRelease struct {
	sync.Once
	os       string
	platform string
}

func getOSInfo() (osName, platform string) {
	cachedOSRelease.Do(func() {
		f, err := os.Open("/etc/os-release")
		if err != nil {
			return
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		name, version := "", ""
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "NAME=") {
				name = strings.Trim(strings.TrimPrefix(line, "NAME="), `"`)
			} else if strings.HasPrefix(line, "VERSION_ID=") {
				version = strings.Trim(strings.TrimPrefix(line, "VERSION_ID="), `"`)
			}
		}
		cachedOSRelease.os = strings.ToLower(name)
		if name != "" && version != "" {
			cachedOSRelease.platform = fmt.Sprintf("%s %s", name, version)
		} else {
			cachedOSRelease.platform = name
		}
	})
	return cachedOSRelease.os, cachedOSRelease.platform
}

// ─── cached total memory ─────────────────────────────────────────────────────

var cachedTotalMem struct {
	mu        sync.RWMutex
	total     uint64
	expiresAt time.Time
}

const totalMemCacheTTL = 5 * time.Minute

// getTotalMem returns the system total physical memory, refreshing every 5 minutes.
func getTotalMem(now time.Time) uint64 {
	cachedTotalMem.mu.RLock()
	if cachedTotalMem.total > 0 && now.Before(cachedTotalMem.expiresAt) {
		v := cachedTotalMem.total
		cachedTotalMem.mu.RUnlock()
		return v
	}
	cachedTotalMem.mu.RUnlock()

	ms, ok := readMemInfo()
	if !ok {
		return 0
	}
	cachedTotalMem.mu.Lock()
	cachedTotalMem.total = ms.MemTotal
	cachedTotalMem.expiresAt = now.Add(totalMemCacheTTL)
	cachedTotalMem.mu.Unlock()
	return ms.MemTotal
}

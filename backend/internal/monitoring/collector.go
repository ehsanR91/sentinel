package monitoring

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

// SystemSnapshot holds a point-in-time capture of all system metrics.
type SystemSnapshot struct {
	// CPU
	CPUPct   float64   `json:"cpu_pct"`
	CPUCores []float64 `json:"cpu_cores"`

	// Memory
	RAMPct   float64 `json:"ram_pct"`
	RAMUsed  uint64  `json:"ram_used"`
	RAMTotal uint64  `json:"ram_total"`

	// Swap
	SwapPct   float64 `json:"swap_pct"`
	SwapUsed  uint64  `json:"swap_used"`
	SwapTotal uint64  `json:"swap_total"`

	// Disk (root partition)
	DiskPct   float64 `json:"disk_pct"`
	DiskUsed  uint64  `json:"disk_used"`
	DiskTotal uint64  `json:"disk_total"`
	DiskFree  uint64  `json:"disk_free"`

	// All disk partitions
	Partitions []DiskPartition `json:"partitions"`

	// Network (aggregate, bytes/sec since last sample)
	NetRxRate  float64 `json:"net_rx_rate"`
	NetTxRate  float64 `json:"net_tx_rate"`
	NetRxTotal uint64  `json:"net_rx_total"`
	NetTxTotal uint64  `json:"net_tx_total"`

	// Load
	Load1  float64 `json:"load1"`
	Load5  float64 `json:"load5"`
	Load15 float64 `json:"load15"`

	// Host
	Uptime       uint64 `json:"uptime"`
	Hostname     string `json:"hostname"`
	OS           string `json:"os"`
	Kernel       string `json:"kernel"`
	Platform     string `json:"platform"`
	UnreadAlerts int    `json:"unread_alerts"`
	ActiveBans   int    `json:"active_bans"`

	Timestamp int64 `json:"ts"`
}

type DiskPartition struct {
	Mountpoint string  `json:"mount"`
	Device     string  `json:"device"`
	Fstype     string  `json:"fstype"`
	Total      uint64  `json:"total"`
	Used       uint64  `json:"used"`
	Free       uint64  `json:"free"`
	UsedPct    float64 `json:"pct"`
}

// Collector gathers system metrics on a tick interval.
type Collector struct {
	mu          sync.RWMutex
	latest      *SystemSnapshot
	prevNetRx   uint64
	prevNetTx   uint64
	prevNetTime time.Time
	partitions  []DiskPartition
	partsAt     time.Time
	slowEvery   time.Duration
	procMu      sync.RWMutex
	topProcs    []ProcessInfo
	netProcs    []NetworkProcessInfo
	suspicious  []SuspiciousProcess
	procCPUPrev map[int32]processCPUSample
	procEvery   time.Duration
	auxEvery    time.Duration
}

func NewCollector() *Collector {
	return &Collector{}
}

// Start launches the background collection loop.

func (c *Collector) Start(interval, slowEvery time.Duration) {
	if interval <= 0 {
		interval = 2 * time.Second
	}
	if slowEvery < interval {
		slowEvery = interval
	}
	c.slowEvery = slowEvery

	// Warm up CPU delta cache so first snapshot has valid deltas.
	readCPUPercent() //nolint
	c.procEvery = normalizedProcessRefreshInterval(slowEvery)
	c.auxEvery = normalizedAuxiliaryProcessRefreshInterval(c.procEvery)
	c.refreshTopProcessSnapshots()
	c.refreshAuxiliaryProcessSnapshots()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Collect net baseline
		if ns, ok := readNetDevAgg(); ok {
			c.prevNetRx = ns.RxBytes
			c.prevNetTx = ns.TxBytes
			c.prevNetTime = time.Now()
		}

		for range ticker.C {
			snap := c.collect()
			c.mu.Lock()
			c.latest = snap
			c.mu.Unlock()
		}
	}()

	go func() {
		ticker := time.NewTicker(c.procEvery)
		defer ticker.Stop()
		for range ticker.C {
			c.refreshTopProcessSnapshots()
		}
	}()

	go func() {
		ticker := time.NewTicker(c.auxEvery)
		defer ticker.Stop()
		for range ticker.C {
			c.refreshAuxiliaryProcessSnapshots()
		}
	}()
}

// Latest returns the most recent snapshot (safe for concurrent reads).
func (c *Collector) Latest() *SystemSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.latest == nil {
		return &SystemSnapshot{Timestamp: time.Now().Unix()}
	}
	return c.latest
}

func (c *Collector) TopProcesses(limit int) []ProcessInfo {
	items := c.snapshotTopProcesses()
	if len(items) == 0 {
		c.refreshTopProcessSnapshots()
		items = c.snapshotTopProcesses()
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (c *Collector) TopNetworkProcesses(limit int) []NetworkProcessInfo {
	items := c.snapshotNetworkProcesses()
	if len(items) == 0 {
		c.refreshAuxiliaryProcessSnapshots()
		items = c.snapshotNetworkProcesses()
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (c *Collector) SuspiciousProcesses() []SuspiciousProcess {
	items := c.snapshotSuspiciousProcesses()
	if len(items) == 0 {
		c.refreshAuxiliaryProcessSnapshots()
		items = c.snapshotSuspiciousProcesses()
	}
	return items
}

func (c *Collector) collect() *SystemSnapshot {
	now := time.Now()
	snap := &SystemSnapshot{Timestamp: now.Unix()}

	// ── CPU (/proc/stat delta) ────────────────────────────────────────────────
	if perCore, agg, ok := readCPUPercent(); ok {
		snap.CPUPct = agg
		snap.CPUCores = perCore
	}

	// ── Memory (/proc/meminfo) ────────────────────────────────────────────────
	if ms, ok := readMemInfo(); ok {
		used := ms.MemTotal - ms.MemAvailable
		snap.RAMTotal = ms.MemTotal
		snap.RAMUsed = used
		if ms.MemTotal > 0 {
			snap.RAMPct = round2(float64(used) / float64(ms.MemTotal) * 100)
		}
		swapUsed := ms.SwapTotal - ms.SwapFree
		snap.SwapTotal = ms.SwapTotal
		snap.SwapUsed = swapUsed
		if ms.SwapTotal > 0 {
			snap.SwapPct = round2(float64(swapUsed) / float64(ms.SwapTotal) * 100)
		}
	}

	// ── Disk (gopsutil; no procfs equivalent for usage) ───────────────────────
	if du, err := disk.Usage("/"); err == nil {
		snap.DiskPct = round2(du.UsedPercent)
		snap.DiskUsed = du.Used
		snap.DiskTotal = du.Total
		snap.DiskFree = du.Free
	}

	c.refreshPartitions(now)
	c.mu.RLock()
	if len(c.partitions) > 0 {
		snap.Partitions = append([]DiskPartition(nil), c.partitions...)
	}
	c.mu.RUnlock()

	// ── Network (/proc/net/dev) ───────────────────────────────────────────────
	if ns, ok := readNetDevAgg(); ok {
		snap.NetRxTotal = ns.RxBytes
		snap.NetTxTotal = ns.TxBytes
		if !c.prevNetTime.IsZero() {
			elapsed := now.Sub(c.prevNetTime).Seconds()
			if elapsed > 0 {
				snap.NetRxRate = round2(float64(ns.RxBytes-c.prevNetRx) / elapsed)
				snap.NetTxRate = round2(float64(ns.TxBytes-c.prevNetTx) / elapsed)
			}
		}
		c.prevNetRx = ns.RxBytes
		c.prevNetTx = ns.TxBytes
		c.prevNetTime = now
	}

	// ── Load (/proc/loadavg) ──────────────────────────────────────────────────
	if l1, l5, l15, ok := readLoadAvg(); ok {
		snap.Load1 = round2(l1)
		snap.Load5 = round2(l5)
		snap.Load15 = round2(l15)
	}

	// ── Host (cached; procfs-based) ───────────────────────────────────────────
	if uptime, ok := readUptimeSeconds(); ok {
		snap.Uptime = uptime
	}
	snap.Hostname = getHostname()
	snap.Kernel = getKernelVersion()
	snap.OS, snap.Platform = getOSInfo()

	return snap
}

func (c *Collector) refreshPartitions(now time.Time) {
	c.mu.RLock()
	needsRefresh := len(c.partitions) == 0 || now.Sub(c.partsAt) >= c.slowEvery
	c.mu.RUnlock()
	if !needsRefresh {
		return
	}

	parts, err := disk.Partitions(false)
	if err != nil {
		return
	}
	updated := make([]DiskPartition, 0, len(parts))
	for _, p := range parts {
		du, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}
		updated = append(updated, DiskPartition{
			Mountpoint: p.Mountpoint,
			Device:     p.Device,
			Fstype:     p.Fstype,
			Total:      du.Total,
			Used:       du.Used,
			Free:       du.Free,
			UsedPct:    round2(du.UsedPercent),
		})
	}

	c.mu.Lock()
	c.partitions = updated
	c.partsAt = now
	c.mu.Unlock()
}

func (c *Collector) refreshTopProcessSnapshots() {
	bundle := collectProcessSnapshotBundle(c.procCPUPrev, time.Now(), processRefreshOptions{topLimit: processSnapshotLimit})
	c.procMu.Lock()
	c.topProcs = append([]ProcessInfo(nil), bundle.top...)
	c.procCPUPrev = bundle.cpuPrev
	c.procMu.Unlock()
}

func (c *Collector) refreshAuxiliaryProcessSnapshots() {
	bundle := collectProcessSnapshotBundle(c.procCPUPrev, time.Now(), processRefreshOptions{
		includeNetwork:      true,
		networkLimit:        networkProcessSnapshotLimit,
		includeSuspicious:   true,
		includeNetworkTimes: true,
	})
	c.procMu.Lock()
	c.netProcs = append([]NetworkProcessInfo(nil), bundle.network...)
	c.suspicious = append([]SuspiciousProcess(nil), bundle.suspicious...)
	c.procCPUPrev = bundle.cpuPrev
	c.procMu.Unlock()
}

func (c *Collector) snapshotTopProcesses() []ProcessInfo {
	c.procMu.RLock()
	defer c.procMu.RUnlock()
	return append([]ProcessInfo(nil), c.topProcs...)
}

func (c *Collector) snapshotNetworkProcesses() []NetworkProcessInfo {
	c.procMu.RLock()
	defer c.procMu.RUnlock()
	return append([]NetworkProcessInfo(nil), c.netProcs...)
}

func (c *Collector) snapshotSuspiciousProcesses() []SuspiciousProcess {
	c.procMu.RLock()
	defer c.procMu.RUnlock()
	return append([]SuspiciousProcess(nil), c.suspicious...)
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

func normalizedProcessRefreshInterval(refreshEvery time.Duration) time.Duration {
	if refreshEvery <= 0 {
		refreshEvery = 30 * time.Second
	}
	if refreshEvery < 10*time.Second {
		return 10 * time.Second
	}
	return refreshEvery
}

func normalizedAuxiliaryProcessRefreshInterval(procEvery time.Duration) time.Duration {
	if procEvery <= 0 {
		procEvery = 30 * time.Second
	}
	auxEvery := procEvery * 2
	if auxEvery < auxiliaryProcessRefreshMin {
		return auxiliaryProcessRefreshMin
	}
	return auxEvery
}

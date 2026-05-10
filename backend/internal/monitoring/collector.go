package monitoring

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	psnet "github.com/shirou/gopsutil/v3/net"
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
	prevNet     []psnet.IOCountersStat
	prevNetTime time.Time
	hostInfo    host.InfoStat
	hostInfoAt  time.Time
	hostReady   bool
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
	if info, err := host.Info(); err == nil {
		c.hostInfo = *info
		c.hostInfoAt = time.Now()
		c.hostReady = true
	}

	// Warm up: first cpu.Percent call always returns 0
	cpu.Percent(0, true) //nolint
	c.procEvery = normalizedProcessRefreshInterval(slowEvery)
	c.auxEvery = normalizedAuxiliaryProcessRefreshInterval(c.procEvery)
	c.refreshTopProcessSnapshots()
	c.refreshAuxiliaryProcessSnapshots()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		// Collect net baseline
		if nets, err := psnet.IOCounters(false); err == nil {
			c.prevNet = nets
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
	snap := &SystemSnapshot{Timestamp: time.Now().Unix()}

	// ── CPU ──────────────────────────────────────────────────────────────────
	if perCore, err := cpu.Percent(0, true); err == nil {
		var total float64
		for i, p := range perCore {
			_ = i
			snap.CPUCores = append(snap.CPUCores, round2(p))
			total += p
		}
		if len(perCore) > 0 {
			snap.CPUPct = round2(total / float64(len(perCore)))
		}
	}

	// ── Memory ───────────────────────────────────────────────────────────────
	if vm, err := mem.VirtualMemory(); err == nil {
		snap.RAMPct = round2(vm.UsedPercent)
		snap.RAMUsed = vm.Used
		snap.RAMTotal = vm.Total
	}
	if sw, err := mem.SwapMemory(); err == nil {
		snap.SwapPct = round2(sw.UsedPercent)
		snap.SwapUsed = sw.Used
		snap.SwapTotal = sw.Total
	}

	// ── Disk ─────────────────────────────────────────────────────────────────
	if du, err := disk.Usage("/"); err == nil {
		snap.DiskPct = round2(du.UsedPercent)
		snap.DiskUsed = du.Used
		snap.DiskTotal = du.Total
		snap.DiskFree = du.Free
	}

	c.refreshPartitions(time.Now())
	c.mu.RLock()
	if len(c.partitions) > 0 {
		snap.Partitions = append([]DiskPartition(nil), c.partitions...)
	}
	c.mu.RUnlock()

	// ── Network ───────────────────────────────────────────────────────────────
	if nets, err := psnet.IOCounters(false); err == nil && len(nets) > 0 {
		now := time.Now()
		if len(c.prevNet) > 0 {
			elapsed := now.Sub(c.prevNetTime).Seconds()
			if elapsed > 0 {
				snap.NetRxRate = round2(float64(nets[0].BytesRecv-c.prevNet[0].BytesRecv) / elapsed)
				snap.NetTxRate = round2(float64(nets[0].BytesSent-c.prevNet[0].BytesSent) / elapsed)
			}
		}
		snap.NetRxTotal = nets[0].BytesRecv
		snap.NetTxTotal = nets[0].BytesSent
		c.prevNet = nets
		c.prevNetTime = now
	}

	// ── Load ─────────────────────────────────────────────────────────────────
	if avg, err := load.Avg(); err == nil {
		snap.Load1 = round2(avg.Load1)
		snap.Load5 = round2(avg.Load5)
		snap.Load15 = round2(avg.Load15)
	}

	// ── Host ─────────────────────────────────────────────────────────────────
	if c.hostReady {
		snap.Uptime = c.hostInfo.Uptime + uint64(time.Since(c.hostInfoAt).Seconds())
		snap.Hostname = c.hostInfo.Hostname
		snap.OS = c.hostInfo.OS
		snap.Kernel = c.hostInfo.KernelVersion
		snap.Platform = c.hostInfo.Platform + " " + c.hostInfo.PlatformVersion
	} else if info, err := host.Info(); err == nil {
		c.mu.Lock()
		c.hostInfo = *info
		c.hostInfoAt = time.Now()
		c.hostReady = true
		c.mu.Unlock()
		snap.Uptime = info.Uptime
		snap.Hostname = info.Hostname
		snap.OS = info.OS
		snap.Kernel = info.KernelVersion
		snap.Platform = info.Platform + " " + info.PlatformVersion
	}

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

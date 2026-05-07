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
	Uptime   uint64 `json:"uptime"`
	Hostname string `json:"hostname"`
	OS       string `json:"os"`
	Kernel   string `json:"kernel"`
	Platform string `json:"platform"`

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
}

func NewCollector() *Collector {
	return &Collector{}
}

// Start launches the background collection loop.
func (c *Collector) Start(interval time.Duration) {
	// Warm up: first cpu.Percent call always returns 0
	cpu.Percent(0, false) //nolint

	go func() {
		// Collect net baseline
		if nets, err := psnet.IOCounters(false); err == nil {
			c.prevNet = nets
			c.prevNetTime = time.Now()
		}

		for {
			snap := c.collect()
			c.mu.Lock()
			c.latest = snap
			c.mu.Unlock()
			time.Sleep(interval)
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

func (c *Collector) collect() *SystemSnapshot {
	snap := &SystemSnapshot{Timestamp: time.Now().Unix()}

	// ── CPU ──────────────────────────────────────────────────────────────────
	if pcts, err := cpu.Percent(0, false); err == nil && len(pcts) > 0 {
		snap.CPUPct = round2(pcts[0])
	}
	if perCore, err := cpu.Percent(0, true); err == nil {
		for i, p := range perCore {
			_ = i
			snap.CPUCores = append(snap.CPUCores, round2(p))
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

	// All partitions
	if parts, err := disk.Partitions(false); err == nil {
		for _, p := range parts {
			if du, err := disk.Usage(p.Mountpoint); err == nil {
				snap.Partitions = append(snap.Partitions, DiskPartition{
					Mountpoint: p.Mountpoint,
					Device:     p.Device,
					Fstype:     p.Fstype,
					Total:      du.Total,
					Used:       du.Used,
					Free:       du.Free,
					UsedPct:    round2(du.UsedPercent),
				})
			}
		}
	}

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
	if info, err := host.Info(); err == nil {
		snap.Uptime = info.Uptime
		snap.Hostname = info.Hostname
		snap.OS = info.OS
		snap.Kernel = info.KernelVersion
		snap.Platform = info.Platform + " " + info.PlatformVersion
	}

	return snap
}

func round2(f float64) float64 {
	return float64(int(f*100+0.5)) / 100
}

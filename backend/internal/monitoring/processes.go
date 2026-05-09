package monitoring

import (
	"fmt"
	"sort"
	"strings"
	"syscall"
	"time"

	goproc "github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo holds per-process data.
type ProcessInfo struct {
	PID     int32   `json:"pid"`
	Name    string  `json:"name"`
	User    string  `json:"user"`
	CPUPct  float64 `json:"cpu_pct"`
	MemPct  float32 `json:"mem_pct"`
	MemRSS  uint64  `json:"mem_rss"`
	Status  string  `json:"status"`
	Cmdline string  `json:"cmdline"`
}

// NetworkProcessInfo describes a process with active network sockets.
type NetworkProcessInfo struct {
	PID         int32   `json:"pid"`
	Name        string  `json:"name"`
	User        string  `json:"user"`
	CPUPct      float64 `json:"cpu_pct"`
	MemPct      float32 `json:"mem_pct"`
	MemRSS      uint64  `json:"mem_rss"`
	Status      string  `json:"status"`
	Cmdline     string  `json:"cmdline"`
	CreateTime  int64   `json:"create_time"`
	UptimeSec   int64   `json:"uptime_sec"`
	Connections int     `json:"connections"`
	TCP         int     `json:"tcp"`
	UDP         int     `json:"udp"`
	Listen      int     `json:"listen"`
	Established int     `json:"established"`
	RxRate      uint64  `json:"rx_rate"`
	TxRate      uint64  `json:"tx_rate"`
	RateSource  string  `json:"rate_source"`
}

// TopProcesses returns the top N processes sorted by CPU% descending.
func TopProcesses(n int) []ProcessInfo {
	procs, err := goproc.Processes()
	if err != nil {
		return nil
	}

	result := make([]ProcessInfo, 0, len(procs))
	for _, p := range procs {
		name, _ := p.Name()
		cpuPct, _ := p.CPUPercent()
		memPct, _ := p.MemoryPercent()
		memInfo, _ := p.MemoryInfo()
		statuses, _ := p.Status()
		user, _ := p.Username()
		cmd, _ := p.Cmdline()

		status := "sleeping"
		if len(statuses) > 0 {
			status = statuses[0]
		}

		rss := uint64(0)
		if memInfo != nil {
			rss = memInfo.RSS
		}

		if cmd != "" && len(cmd) > 80 {
			cmd = cmd[:80] + "…"
		}

		result = append(result, ProcessInfo{
			PID:     p.Pid,
			Name:    name,
			User:    user,
			CPUPct:  round2(cpuPct),
			MemPct:  float32(int(float64(memPct)*10+0.5)) / 10,
			MemRSS:  rss,
			Status:  status,
			Cmdline: cmd,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CPUPct > result[j].CPUPct
	})

	if len(result) > n {
		return result[:n]
	}
	return result
}

// TopNetworkProcesses returns processes with active network sockets.
func TopNetworkProcesses(n int) []NetworkProcessInfo {
	procs, err := goproc.Processes()
	if err != nil {
		return []NetworkProcessInfo{}
	}

	result := make([]NetworkProcessInfo, 0, len(procs))
	for _, p := range procs {
		conns, err := p.Connections()
		if err != nil || len(conns) == 0 {
			continue
		}

		name, _ := p.Name()
		cpuPct, _ := p.CPUPercent()
		memPct, _ := p.MemoryPercent()
		memInfo, _ := p.MemoryInfo()
		statuses, _ := p.Status()
		user, _ := p.Username()
		cmd, _ := p.Cmdline()
		createTime, _ := p.CreateTime()

		status := "sleeping"
		if len(statuses) > 0 {
			status = statuses[0]
		}
		if cmd != "" && len(cmd) > 80 {
			cmd = cmd[:80] + "…"
		}

		rss := uint64(0)
		if memInfo != nil {
			rss = memInfo.RSS
		}

		info := NetworkProcessInfo{
			PID:         p.Pid,
			Name:        name,
			User:        user,
			CPUPct:      round2(cpuPct),
			MemPct:      float32(int(float64(memPct)*10+0.5)) / 10,
			MemRSS:      rss,
			Status:      status,
			Cmdline:     cmd,
			CreateTime:  createTime,
			UptimeSec:   processUptimeSeconds(createTime),
			Connections: len(conns),
			RateSource:  "socket-count",
		}

		for _, conn := range conns {
			switch conn.Type {
			case syscall.SOCK_STREAM:
				info.TCP++
			case syscall.SOCK_DGRAM:
				info.UDP++
			}
			switch strings.ToLower(conn.Status) {
			case "listen":
				info.Listen++
			case "established":
				info.Established++
			}
		}

		result = append(result, info)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Established != result[j].Established {
			return result[i].Established > result[j].Established
		}
		return result[i].Connections > result[j].Connections
	})

	if len(result) > n {
		return result[:n]
	}
	return result
}

func processUptimeSeconds(createTimeMs int64) int64 {
	if createTimeMs <= 0 {
		return 0
	}
	uptime := time.Since(time.UnixMilli(createTimeMs)).Seconds()
	if uptime < 0 {
		return 0
	}
	return int64(uptime)
}

// SuspiciousProcess describes a process flagged by heuristic detection.
type SuspiciousProcess struct {
	PID    int32  `json:"pid"`
	Name   string `json:"name"`
	User   string `json:"user"`
	Reason string `json:"reason"`
	Risk   string `json:"risk"` // "high" | "medium" | "low"
	Cmd    string `json:"cmd"`
}

var suspiciousPatterns = []struct{ kw, reason, risk string }{
	{"nmap", "Network scanner", "high"},
	{"masscan", "Mass port scanner", "high"},
	{"nikto", "Web vulnerability scanner", "high"},
	{"sqlmap", "SQL injection tool", "high"},
	{"msfconsole", "Metasploit framework", "high"},
	{"metasploit", "Metasploit framework", "high"},
	{"hydra", "Brute-force tool", "high"},
	{"medusa", "Brute-force tool", "high"},
	{"aircrack", "Wireless cracker", "high"},
	{"john", "Password cracker", "medium"},
	{"hashcat", "Password cracker", "medium"},
	{"socat", "Socket relay (possible backdoor)", "medium"},
	{"bash -i", "Interactive shell spawn", "medium"},
	{"sh -i", "Interactive shell spawn", "medium"},
	{"python -c", "Inline Python execution", "medium"},
	{"perl -e", "Inline Perl execution", "medium"},
	{"ruby -e", "Inline Ruby execution", "medium"},
}

// DetectSuspiciousProcesses scans running processes for known-bad patterns
// and excessive network connections, returning flagged entries.
func DetectSuspiciousProcesses() []SuspiciousProcess {
	procs, err := goproc.Processes()
	if err != nil {
		return []SuspiciousProcess{}
	}

	seen := map[int32]bool{}
	var results []SuspiciousProcess

	for _, p := range procs {
		name, _ := p.Name()
		cmd, _ := p.Cmdline()
		user, _ := p.Username()

		lower := strings.ToLower(name + " " + cmd)

		for _, pat := range suspiciousPatterns {
			if strings.Contains(lower, pat.kw) {
				if !seen[p.Pid] {
					seen[p.Pid] = true
					short := cmd
					if len(short) > 120 {
						short = short[:120] + "…"
					}
					results = append(results, SuspiciousProcess{
						PID:    p.Pid,
						Name:   name,
						User:   user,
						Reason: pat.reason,
						Risk:   pat.risk,
						Cmd:    short,
					})
				}
				break
			}
		}

		// Flag processes with an unusually high number of network connections.
		if !seen[p.Pid] {
			conns, err := p.Connections()
			if err == nil && len(conns) > 50 {
				seen[p.Pid] = true
				short := cmd
				if len(short) > 120 {
					short = short[:120] + "…"
				}
				results = append(results, SuspiciousProcess{
					PID:    p.Pid,
					Name:   name,
					User:   user,
					Reason: fmt.Sprintf("Excessive network connections: %d", len(conns)),
					Risk:   "medium",
					Cmd:    short,
				})
			}
		}
	}

	if results == nil {
		return []SuspiciousProcess{}
	}
	return results
}

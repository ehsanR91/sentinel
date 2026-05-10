package monitoring

import (
	"fmt"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/mem"
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

const (
	processSnapshotLimit        = 100
	networkProcessSnapshotLimit = 100
	suspiciousConnectionCount   = 50
)

type processCPUSample struct {
	total  float64
	seenAt time.Time
}

type processSnapshotBundle struct {
	top        []ProcessInfo
	network    []NetworkProcessInfo
	suspicious []SuspiciousProcess
	cpuPrev    map[int32]processCPUSample
}

type processCandidate struct {
	pid         int32
	name        string
	cpuPct      float64
	connections int
	tcp         int
	udp         int
	listen      int
	established int
	risk        string
	reason      string
	cmdline     string
}

type processDetail struct {
	user       string
	status     string
	cmdline    string
	memRSS     uint64
	memPct     float32
	createTime int64
}

func collectProcessSnapshotBundle(prev map[int32]processCPUSample, now time.Time, topLimit, networkLimit int) processSnapshotBundle {
	if now.IsZero() {
		now = time.Now()
	}
	if topLimit <= 0 {
		topLimit = processSnapshotLimit
	}
	if networkLimit <= 0 {
		networkLimit = networkProcessSnapshotLimit
	}

	procs, err := goproc.Processes()
	if err != nil {
		return processSnapshotBundle{
			top:        []ProcessInfo{},
			network:    []NetworkProcessInfo{},
			suspicious: []SuspiciousProcess{},
			cpuPrev:    map[int32]processCPUSample{},
		}
	}

	totalMem := uint64(0)
	if vm, err := mem.VirtualMemory(); err == nil {
		totalMem = vm.Total
	}

	procByPID := make(map[int32]*goproc.Process, len(procs))
	nextPrev := make(map[int32]processCPUSample, len(procs))
	topCandidates := make([]processCandidate, 0, len(procs))
	networkCandidates := make([]processCandidate, 0, len(procs)/2)
	suspiciousCandidates := make([]processCandidate, 0, 8)

	for _, p := range procs {
		procByPID[p.Pid] = p

		name, _ := p.Name()
		if name == "" {
			name = strconv.FormatInt(int64(p.Pid), 10)
		}

		sample, cpuPct, ok := sampleProcessCPU(p, prev, now)
		if ok {
			nextPrev[p.Pid] = sample
		}

		candidate := processCandidate{
			pid:    p.Pid,
			name:   name,
			cpuPct: cpuPct,
		}

		cmdline := ""
		if needsCmdlineInspection(name) {
			cmdline, _ = p.Cmdline()
		}
		if reason, risk := suspiciousMatch(name, cmdline); reason != "" {
			candidate.reason = reason
			candidate.risk = risk
			candidate.cmdline = cmdline
		}

		conns, err := p.Connections()
		if err == nil && len(conns) > 0 {
			candidate.connections = len(conns)
			for _, conn := range conns {
				switch conn.Type {
				case syscall.SOCK_STREAM:
					candidate.tcp++
				case syscall.SOCK_DGRAM:
					candidate.udp++
				}
				switch strings.ToLower(conn.Status) {
				case "listen":
					candidate.listen++
				case "established":
					candidate.established++
				}
			}
			networkCandidates = append(networkCandidates, candidate)
			if candidate.reason == "" && candidate.connections > suspiciousConnectionCount {
				candidate.reason = fmt.Sprintf("Excessive network connections: %d", candidate.connections)
				candidate.risk = "medium"
			}
		}

		topCandidates = append(topCandidates, candidate)
		if candidate.reason != "" {
			suspiciousCandidates = append(suspiciousCandidates, candidate)
		}
	}

	sort.Slice(topCandidates, func(i, j int) bool {
		if topCandidates[i].cpuPct == topCandidates[j].cpuPct {
			return topCandidates[i].pid < topCandidates[j].pid
		}
		return topCandidates[i].cpuPct > topCandidates[j].cpuPct
	})
	sort.Slice(networkCandidates, func(i, j int) bool {
		if networkCandidates[i].established != networkCandidates[j].established {
			return networkCandidates[i].established > networkCandidates[j].established
		}
		if networkCandidates[i].connections != networkCandidates[j].connections {
			return networkCandidates[i].connections > networkCandidates[j].connections
		}
		return networkCandidates[i].cpuPct > networkCandidates[j].cpuPct
	})

	if len(topCandidates) > topLimit {
		topCandidates = topCandidates[:topLimit]
	}
	if len(networkCandidates) > networkLimit {
		networkCandidates = networkCandidates[:networkLimit]
	}

	userCache := map[uint32]string{}
	detailCache := map[int32]processDetail{}
	top := make([]ProcessInfo, 0, len(topCandidates))
	for _, candidate := range topCandidates {
		detail := loadProcessDetail(procByPID[candidate.pid], totalMem, userCache, detailCache, true, false)
		top = append(top, ProcessInfo{
			PID:     candidate.pid,
			Name:    candidate.name,
			User:    detail.user,
			CPUPct:  candidate.cpuPct,
			MemPct:  detail.memPct,
			MemRSS:  detail.memRSS,
			Status:  detail.status,
			Cmdline: detail.cmdline,
		})
	}

	network := make([]NetworkProcessInfo, 0, len(networkCandidates))
	for _, candidate := range networkCandidates {
		detail := loadProcessDetail(procByPID[candidate.pid], totalMem, userCache, detailCache, true, true)
		network = append(network, NetworkProcessInfo{
			PID:         candidate.pid,
			Name:        candidate.name,
			User:        detail.user,
			CPUPct:      candidate.cpuPct,
			MemPct:      detail.memPct,
			MemRSS:      detail.memRSS,
			Status:      detail.status,
			Cmdline:     detail.cmdline,
			CreateTime:  detail.createTime,
			UptimeSec:   processUptimeSeconds(detail.createTime),
			Connections: candidate.connections,
			TCP:         candidate.tcp,
			UDP:         candidate.udp,
			Listen:      candidate.listen,
			Established: candidate.established,
			RateSource:  "socket-count",
		})
	}

	suspicious := make([]SuspiciousProcess, 0, len(suspiciousCandidates))
	seen := make(map[int32]bool, len(suspiciousCandidates))
	for _, candidate := range suspiciousCandidates {
		if seen[candidate.pid] {
			continue
		}
		seen[candidate.pid] = true
		detail := loadProcessDetail(procByPID[candidate.pid], totalMem, userCache, detailCache, true, false)
		cmdline := candidate.cmdline
		if cmdline == "" {
			cmdline = detail.cmdline
		}
		suspicious = append(suspicious, SuspiciousProcess{
			PID:    candidate.pid,
			Name:   candidate.name,
			User:   detail.user,
			Reason: candidate.reason,
			Risk:   candidate.risk,
			Cmd:    truncateProcessText(cmdline, 120),
		})
	}

	return processSnapshotBundle{
		top:        top,
		network:    network,
		suspicious: suspicious,
		cpuPrev:    nextPrev,
	}
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

func sampleProcessCPU(p *goproc.Process, prev map[int32]processCPUSample, now time.Time) (processCPUSample, float64, bool) {
	times, err := p.Times()
	if err != nil || times == nil {
		return processCPUSample{}, 0, false
	}
	total := times.User + times.System
	sample := processCPUSample{total: total, seenAt: now}
	last, ok := prev[p.Pid]
	if !ok || last.seenAt.IsZero() || total < last.total {
		return sample, 0, true
	}
	elapsed := now.Sub(last.seenAt).Seconds()
	if elapsed <= 0 {
		return sample, 0, true
	}
	return sample, round2(((total - last.total) / elapsed) * 100), true
}

func suspiciousMatch(name, cmdline string) (string, string) {
	lower := strings.ToLower(strings.TrimSpace(name))
	for _, pat := range suspiciousPatterns {
		if strings.Contains(lower, pat.kw) {
			return pat.reason, pat.risk
		}
	}
	if cmdline == "" {
		return "", ""
	}
	lower = strings.ToLower(name + " " + cmdline)
	for _, pat := range suspiciousPatterns {
		if strings.Contains(lower, pat.kw) {
			return pat.reason, pat.risk
		}
	}
	return "", ""
}

func needsCmdlineInspection(name string) bool {
	lower := strings.ToLower(strings.TrimSpace(name))
	switch lower {
	case "bash", "sh", "python", "python3", "perl", "ruby", "socat":
		return true
	default:
		return false
	}
}

func loadProcessDetail(
	p *goproc.Process,
	totalMem uint64,
	userCache map[uint32]string,
	detailCache map[int32]processDetail,
	needCmdline bool,
	needCreateTime bool,
) processDetail {
	if p == nil {
		return processDetail{}
	}
	if cached, ok := detailCache[p.Pid]; ok {
		if (!needCmdline || cached.cmdline != "") && (!needCreateTime || cached.createTime != 0) {
			return cached
		}
	}

	detail := detailCache[p.Pid]
	if detail.user == "" {
		detail.user = resolveProcessUser(p, userCache)
	}
	if detail.status == "" {
		statuses, _ := p.Status()
		detail.status = "sleeping"
		if len(statuses) > 0 {
			detail.status = statuses[0]
		}
	}
	if detail.memRSS == 0 {
		if memInfo, err := p.MemoryInfo(); err == nil && memInfo != nil {
			detail.memRSS = memInfo.RSS
			if totalMem > 0 {
				memPct := (float64(memInfo.RSS) / float64(totalMem)) * 100
				detail.memPct = float32(int(memPct*10+0.5)) / 10
			}
		}
	}
	if needCmdline && detail.cmdline == "" {
		cmdline, _ := p.Cmdline()
		detail.cmdline = truncateProcessText(cmdline, 80)
	}
	if needCreateTime && detail.createTime == 0 {
		detail.createTime, _ = p.CreateTime()
	}
	detailCache[p.Pid] = detail
	return detail
}

func resolveProcessUser(p *goproc.Process, userCache map[uint32]string) string {
	uids, err := p.Uids()
	if err != nil || len(uids) == 0 {
		return ""
	}
	uid := uint32(uids[0])
	if cached, ok := userCache[uid]; ok {
		return cached
	}
	username := strconv.FormatUint(uint64(uid), 10)
	if resolved, err := user.LookupId(username); err == nil && resolved != nil && resolved.Username != "" {
		username = resolved.Username
	}
	userCache[uid] = username
	return username
}

func truncateProcessText(value string, limit int) string {
	if value == "" || limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit] + "…"
}

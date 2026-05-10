package monitoring

import (
	"bufio"
	"fmt"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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
	procStatClockTicks          = 100
	bootTimeCacheTTL            = 10 * time.Minute
	auxiliaryProcessRefreshMin  = 60 * time.Second
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

type processRefreshOptions struct {
	topLimit            int
	networkLimit        int
	includeNetwork      bool
	includeSuspicious   bool
	includeNetworkTimes bool
}

type processCandidate struct {
	pid         int32
	name        string
	cpuPct      float64
	createTime  int64
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
}

type procStatSample struct {
	name            string
	totalCPUSeconds float64
	createTimeMs    int64
}

type cachedBootTime struct {
	mu        sync.RWMutex
	bootTime  int64
	expiresAt time.Time
}

var procBootTimeCache cachedBootTime

func collectProcessSnapshotBundle(prev map[int32]processCPUSample, now time.Time, opts processRefreshOptions) processSnapshotBundle {
	if now.IsZero() {
		now = time.Now()
	}
	if opts.topLimit < 0 {
		opts.topLimit = 0
	}
	if opts.topLimit == 0 && !opts.includeNetwork && !opts.includeSuspicious {
		return processSnapshotBundle{cpuPrev: appendProcessCPUSamples(nil, prev)}
	}
	if opts.topLimit > 0 && opts.topLimit < processSnapshotLimit {
		// keep provided limit
	} else if opts.topLimit > 0 {
		opts.topLimit = processSnapshotLimit
	}
	if opts.includeNetwork {
		if opts.networkLimit <= 0 || opts.networkLimit > networkProcessSnapshotLimit {
			opts.networkLimit = networkProcessSnapshotLimit
		}
	}
	if opts.topLimit > 0 && !opts.includeNetwork && !opts.includeSuspicious {
		return collectTopProcessSnapshotBundle(prev, now, opts.topLimit)
	}

	pids, err := listProcPIDs()
	if err != nil {
		return processSnapshotBundle{
			top:        []ProcessInfo{},
			network:    []NetworkProcessInfo{},
			suspicious: []SuspiciousProcess{},
			cpuPrev:    map[int32]processCPUSample{},
		}
	}

	totalMem := getTotalMem(now)

	var inodeMap map[uint64]procNetSocket
	if opts.includeNetwork || opts.includeSuspicious {
		inodeMap = buildNetInodeMap()
	}

	nextPrev := make(map[int32]processCPUSample, len(pids))
	topCandidates := make([]processCandidate, 0, len(pids))
	networkCandidates := make([]processCandidate, 0, len(pids)/2)
	suspiciousCandidates := make([]processCandidate, 0, 8)

	for _, pid := range pids {
		statSample, statErr := readProcStatSample(pid, now, opts.includeNetworkTimes)
		name := statSample.name
		if name == "" {
			name = strconv.FormatInt(int64(pid), 10)
		}

		sample, cpuPct, ok := sampleProcessCPU(pid, statSample, prev, now, statErr == nil)
		if ok {
			nextPrev[pid] = sample
		}

		candidate := processCandidate{
			pid:        pid,
			name:       name,
			cpuPct:     cpuPct,
			createTime: statSample.createTimeMs,
		}

		cmdline := ""
		if needsCmdlineInspection(name) {
			cmdline = readProcCmdline(pid)
		}
		if reason, risk := suspiciousMatch(name, cmdline); reason != "" {
			candidate.reason = reason
			candidate.risk = risk
			candidate.cmdline = cmdline
		}

		if inodeMap != nil {
			total, tcpCount, udpCount, listenCount, estCount := countProcSockets(pid, inodeMap)
			if total > 0 {
				candidate.connections = total
				candidate.tcp = tcpCount
				candidate.udp = udpCount
				candidate.listen = listenCount
				candidate.established = estCount
				if opts.includeNetwork {
					networkCandidates = append(networkCandidates, candidate)
				}
				if candidate.reason == "" && total > suspiciousConnectionCount {
					candidate.reason = fmt.Sprintf("Excessive network connections: %d", total)
					candidate.risk = "medium"
				}
			}
		}

		if opts.topLimit > 0 {
			topCandidates = append(topCandidates, candidate)
		}
		if opts.includeSuspicious && candidate.reason != "" {
			suspiciousCandidates = append(suspiciousCandidates, candidate)
		}
	}

	if len(topCandidates) > 0 {
		sort.Slice(topCandidates, func(i, j int) bool {
			if topCandidates[i].cpuPct == topCandidates[j].cpuPct {
				return topCandidates[i].pid < topCandidates[j].pid
			}
			return topCandidates[i].cpuPct > topCandidates[j].cpuPct
		})
	}
	if len(networkCandidates) > 0 {
		sort.Slice(networkCandidates, func(i, j int) bool {
			if networkCandidates[i].established != networkCandidates[j].established {
				return networkCandidates[i].established > networkCandidates[j].established
			}
			if networkCandidates[i].connections != networkCandidates[j].connections {
				return networkCandidates[i].connections > networkCandidates[j].connections
			}
			return networkCandidates[i].cpuPct > networkCandidates[j].cpuPct
		})
	}

	if opts.topLimit > 0 && len(topCandidates) > opts.topLimit {
		topCandidates = topCandidates[:opts.topLimit]
	}
	if opts.includeNetwork && len(networkCandidates) > opts.networkLimit {
		networkCandidates = networkCandidates[:opts.networkLimit]
	}

	userCache := map[uint32]string{}
	detailCache := map[int32]processDetail{}
	top := make([]ProcessInfo, 0, len(topCandidates))
	for _, candidate := range topCandidates {
		detail := loadProcessDetail(candidate.pid, totalMem, userCache, detailCache, false)
		top = append(top, ProcessInfo{
			PID:     candidate.pid,
			Name:    candidate.name,
			User:    detail.user,
			CPUPct:  candidate.cpuPct,
			MemPct:  detail.memPct,
			MemRSS:  detail.memRSS,
			Status:  detail.status,
			Cmdline: "",
		})
	}

	network := make([]NetworkProcessInfo, 0, len(networkCandidates))
	for _, candidate := range networkCandidates {
		detail := loadProcessDetail(candidate.pid, totalMem, userCache, detailCache, false)
		createTime := int64(0)
		uptimeSec := int64(0)
		if opts.includeNetworkTimes {
			createTime = candidate.createTime
			uptimeSec = processUptimeSeconds(candidate.createTime)
		}
		network = append(network, NetworkProcessInfo{
			PID:         candidate.pid,
			Name:        candidate.name,
			User:        detail.user,
			CPUPct:      candidate.cpuPct,
			MemPct:      detail.memPct,
			MemRSS:      detail.memRSS,
			Status:      detail.status,
			Cmdline:     "",
			CreateTime:  createTime,
			UptimeSec:   uptimeSec,
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
		detail := loadProcessDetail(candidate.pid, totalMem, userCache, detailCache, true)
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

func appendProcessCPUSamples(dst, src map[int32]processCPUSample) map[int32]processCPUSample {
	if len(src) == 0 {
		return map[int32]processCPUSample{}
	}
	if dst == nil {
		dst = make(map[int32]processCPUSample, len(src))
	}
	for pid, sample := range src {
		dst[pid] = sample
	}
	return dst
}

func collectTopProcessSnapshotBundle(prev map[int32]processCPUSample, now time.Time, topLimit int) processSnapshotBundle {
	if topLimit <= 0 || topLimit > processSnapshotLimit {
		topLimit = processSnapshotLimit
	}
	pids, err := listProcPIDs()
	if err != nil {
		return processSnapshotBundle{top: []ProcessInfo{}, cpuPrev: map[int32]processCPUSample{}}
	}

	totalMem := getTotalMem(now)

	nextPrev := make(map[int32]processCPUSample, len(pids))
	topCandidates := make([]processCandidate, 0, len(pids))
	for _, pid := range pids {
		statSample, err := readProcStatSample(pid, now, false)
		if err != nil {
			continue
		}
		sample, cpuPct, ok := sampleProcessCPU(pid, statSample, prev, now, true)
		if ok {
			nextPrev[pid] = sample
		}
		name := statSample.name
		if name == "" {
			name = strconv.FormatInt(int64(pid), 10)
		}
		topCandidates = append(topCandidates, processCandidate{
			pid:    pid,
			name:   name,
			cpuPct: cpuPct,
		})
	}

	if len(topCandidates) > 0 {
		sort.Slice(topCandidates, func(i, j int) bool {
			if topCandidates[i].cpuPct == topCandidates[j].cpuPct {
				return topCandidates[i].pid < topCandidates[j].pid
			}
			return topCandidates[i].cpuPct > topCandidates[j].cpuPct
		})
	}
	if len(topCandidates) > topLimit {
		topCandidates = topCandidates[:topLimit]
	}

	userCache := map[uint32]string{}
	detailCache := map[int32]processDetail{}
	top := make([]ProcessInfo, 0, len(topCandidates))
	for _, candidate := range topCandidates {
		detail := loadProcessDetail(candidate.pid, totalMem, userCache, detailCache, false)
		top = append(top, ProcessInfo{
			PID:     candidate.pid,
			Name:    candidate.name,
			User:    detail.user,
			CPUPct:  candidate.cpuPct,
			MemPct:  detail.memPct,
			MemRSS:  detail.memRSS,
			Status:  detail.status,
			Cmdline: "",
		})
	}

	return processSnapshotBundle{
		top:     top,
		cpuPrev: nextPrev,
	}
}

func listProcPIDs() ([]int32, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	pids := make([]int32, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.ParseInt(entry.Name(), 10, 32)
		if err != nil || pid <= 0 {
			continue
		}
		pids = append(pids, int32(pid))
	}
	return pids, nil
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

func sampleProcessCPU(pid int32, stat procStatSample, prev map[int32]processCPUSample, now time.Time, hasStat bool) (processCPUSample, float64, bool) {
	if !hasStat {
		return processCPUSample{}, 0, false
	}
	total := stat.totalCPUSeconds
	sample := processCPUSample{total: total, seenAt: now}
	last, ok := prev[pid]
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
	pid int32,
	totalMem uint64,
	userCache map[uint32]string,
	detailCache map[int32]processDetail,
	needCmdline bool,
) processDetail {
	if pid <= 0 {
		return processDetail{}
	}
	if cached, ok := detailCache[pid]; ok {
		if !needCmdline || cached.cmdline != "" {
			return cached
		}
	}

	detail := detailCache[pid]
	if detail.user == "" || detail.status == "" || detail.memRSS == 0 {
		uid, state, rssKB := readProcStatusInfo(pid)
		if detail.status == "" {
			detail.status = state
			if detail.status == "" {
				detail.status = "sleeping"
			}
		}
		if detail.memRSS == 0 && rssKB > 0 {
			detail.memRSS = rssKB * 1024
			if totalMem > 0 {
				memPct := (float64(detail.memRSS) / float64(totalMem)) * 100
				detail.memPct = float32(int(memPct*10+0.5)) / 10
			}
		}
		if detail.user == "" {
			detail.user = resolveUID(uid, userCache)
		}
	}
	if needCmdline && detail.cmdline == "" {
		detail.cmdline = truncateProcessText(readProcCmdline(pid), 80)
	}
	detailCache[pid] = detail
	return detail
}

func readProcStatSample(pid int32, now time.Time, includeCreateTime bool) (procStatSample, error) {
	path := fmt.Sprintf("/proc/%d/stat", pid)
	f, err := os.Open(path)
	if err != nil {
		return procStatSample{}, err
	}
	var buf [512]byte
	n, err := f.Read(buf[:])
	f.Close()
	if err != nil && n == 0 {
		return procStatSample{}, err
	}
	line := strings.TrimSpace(string(buf[:n]))
	openIdx := strings.IndexByte(line, '(')
	closeIdx := strings.LastIndexByte(line, ')')
	if openIdx < 0 || closeIdx <= openIdx {
		return procStatSample{}, fmt.Errorf("unexpected proc stat format")
	}
	name := line[openIdx+1 : closeIdx]
	fields := strings.Fields(strings.TrimSpace(line[closeIdx+1:]))
	if len(fields) < 20 {
		return procStatSample{}, fmt.Errorf("unexpected proc stat field count")
	}
	utimeTicks, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return procStatSample{}, err
	}
	stimeTicks, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return procStatSample{}, err
	}
	startTicks, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return procStatSample{}, err
	}
	createTimeMs := int64(0)
	if includeCreateTime {
		if bootTimeSec := cachedBootTimeSeconds(now); bootTimeSec > 0 {
			createTimeMs = (bootTimeSec * 1000) + int64((float64(startTicks)/float64(procStatClockTicks))*1000)
		}
	}
	return procStatSample{
		name:            name,
		totalCPUSeconds: float64(utimeTicks+stimeTicks) / float64(procStatClockTicks),
		createTimeMs:    createTimeMs,
	}, nil
}

func cachedBootTimeSeconds(now time.Time) int64 {
	procBootTimeCache.mu.RLock()
	if procBootTimeCache.bootTime > 0 && now.Before(procBootTimeCache.expiresAt) {
		bootTime := procBootTimeCache.bootTime
		procBootTimeCache.mu.RUnlock()
		return bootTime
	}
	procBootTimeCache.mu.RUnlock()

	raw, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0
	}
	scanner := bufio.NewScanner(strings.NewReader(string(raw)))
	bootTime := int64(0)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "btime ") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "btime "))
		bootTime, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0
		}
		break
	}
	if bootTime <= 0 {
		return 0
	}
	procBootTimeCache.mu.Lock()
	procBootTimeCache.bootTime = bootTime
	procBootTimeCache.expiresAt = now.Add(bootTimeCacheTTL)
	procBootTimeCache.mu.Unlock()
	return bootTime
}

func resolveUID(uid uint32, userCache map[uint32]string) string {
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

// readProcStatusInfo reads /proc/<pid>/status once and extracts UID, state, and VmRSS.
// Avoids the overhead of gopsutil's locking/reflection machinery.
func readProcStatusInfo(pid int32) (uid uint32, state string, rssKB uint64) {
	path := fmt.Sprintf("/proc/%d/status", pid)
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	got := 0
	for scanner.Scan() && got < 3 {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "State:"):
			// "State:\tR (running)" → take second field (the single letter code)
			if parts := strings.Fields(line); len(parts) >= 2 {
				state = parts[1]
			}
			got++
		case strings.HasPrefix(line, "Uid:"):
			// "Uid:\t1000\t1000\t1000\t1000" → real uid is second field
			if parts := strings.Fields(line); len(parts) >= 2 {
				if v, err := strconv.ParseUint(parts[1], 10, 32); err == nil {
					uid = uint32(v)
				}
			}
			got++
		case strings.HasPrefix(line, "VmRSS:"):
			// "VmRSS:\t12345 kB"
			if parts := strings.Fields(line); len(parts) >= 2 {
				rssKB, _ = strconv.ParseUint(parts[1], 10, 64)
			}
			got++
		}
	}
	return
}

// readProcCmdline reads /proc/<pid>/cmdline and returns a printable string.
func readProcCmdline(pid int32) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil || len(data) == 0 {
		return ""
	}
	// cmdline uses null bytes as argument separators; replace with spaces.
	for i, b := range data {
		if b == 0 {
			data[i] = ' '
		}
	}
	return strings.TrimSpace(string(data))
}

// procNetSocket describes a socket entry from /proc/net/tcp[6] or udp[6].
type procNetSocket struct {
	isTCP         bool
	isListen      bool
	isEstablished bool
}

// buildNetInodeMap reads /proc/net/tcp, /proc/net/tcp6, /proc/net/udp, /proc/net/udp6
// once and returns a map from socket inode to socket metadata.
func buildNetInodeMap() map[uint64]procNetSocket {
	m := make(map[uint64]procNetSocket, 512)
	addNetProtoFile(m, "/proc/net/tcp", true)
	addNetProtoFile(m, "/proc/net/tcp6", true)
	addNetProtoFile(m, "/proc/net/udp", false)
	addNetProtoFile(m, "/proc/net/udp6", false)
	return m
}

func addNetProtoFile(m map[uint64]procNetSocket, path string, isTCP bool) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 256*1024)
	scanner.Scan() // skip header line
	for scanner.Scan() {
		// Fields: sl local_addr rem_addr state tx:rx ... inode ...
		fields := strings.Fields(scanner.Text())
		if len(fields) < 10 {
			continue
		}
		inode, err := strconv.ParseUint(fields[9], 10, 64)
		if err != nil || inode == 0 {
			continue
		}
		info := procNetSocket{isTCP: isTCP}
		if isTCP {
			stateHex, _ := strconv.ParseUint(fields[3], 16, 8)
			switch stateHex {
			case 0x01: // TCP_ESTABLISHED
				info.isEstablished = true
			case 0x0A: // TCP_LISTEN
				info.isListen = true
			}
		}
		m[inode] = info
	}
}

// countProcSockets counts socket file descriptors for a process using
// a pre-built inode map. Reads /proc/<pid>/fd/ once via readlink.
func countProcSockets(pid int32, inodeMap map[uint64]procNetSocket) (total, tcp, udp, listen, established int) {
	fdPath := fmt.Sprintf("/proc/%d/fd", pid)
	entries, err := os.ReadDir(fdPath)
	if err != nil {
		return
	}
	for _, e := range entries {
		link, err := os.Readlink(fdPath + "/" + e.Name())
		if err != nil || len(link) < 10 || link[0] != 's' {
			continue
		}
		if !strings.HasPrefix(link, "socket:[") {
			continue
		}
		// parse "socket:[inode]"
		inodeStr := link[8 : len(link)-1]
		inode, err := strconv.ParseUint(inodeStr, 10, 64)
		if err != nil {
			continue
		}
		info, ok := inodeMap[inode]
		if !ok {
			continue
		}
		total++
		if info.isTCP {
			tcp++
			if info.isListen {
				listen++
			} else if info.isEstablished {
				established++
			}
		} else {
			udp++
		}
	}
	return
}

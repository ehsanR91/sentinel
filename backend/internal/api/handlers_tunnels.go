package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// TunnelableApp represents a detected web-accessible app running on the server.
type TunnelableApp struct {
	Name        string `json:"name"`
	Port        int    `json:"port"`
	Icon        string `json:"icon"`
	Color       string `json:"color"`
	Category    string `json:"category"`
	Source      string `json:"source"` // "process", "docker", "config"
	ProcessName string `json:"process_name,omitempty"`
}

type TunnelConnectionInfo struct {
	Host          string `json:"host"`
	SSHPort       int    `json:"ssh_port"`
	SSHUser       string `json:"ssh_user"`
	SSHUserSource string `json:"ssh_user_source"`
}

type TunnelableAppsResponse struct {
	Apps       []TunnelableApp      `json:"apps"`
	Connection TunnelConnectionInfo `json:"connection"`
}

type grantTunnelAccessRequest struct {
	Port          int `json:"port"`
	DurationHours int `json:"duration_hours"`
}

// ─── Persistent grant store ───────────────────────────────────────────────────

// GrantRecord is one active temporary-access entry stored on disk.
type GrantRecord struct {
	IP         string    `json:"ip"`
	Port       int       `json:"port"`
	Comment    string    `json:"comment"`
	ExpiresAt  time.Time `json:"expires_at"`
	Mode       string    `json:"mode"` // "proxy", "nat", or "ufw"
	Iface      string    `json:"iface,omitempty"`
	ListenHost string    `json:"listen_host,omitempty"`
}

type grantStore struct {
	mu      sync.Mutex
	path    string
	records []GrantRecord
}

func newGrantStore() *grantStore {
	path := grantStorePath()
	if dir := filepath.Dir(path); dir != "." {
		_ = os.MkdirAll(dir, 0o700)
	}
	return &grantStore{path: path}
}

func grantStorePath() string {
	const preferred = "/var/lib/sentinelcore/grants.json"
	if err := os.MkdirAll(filepath.Dir(preferred), 0o700); err == nil {
		return preferred
	}
	// Fallback: next to the binary
	exe, err := os.Executable()
	if err != nil {
		return "grants.json"
	}
	return filepath.Join(filepath.Dir(exe), "grants.json")
}

func (gs *grantStore) load() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	data, err := os.ReadFile(gs.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("[grants] could not read store %s: %v", gs.path, err)
		}
		return
	}
	var recs []GrantRecord
	if err := json.Unmarshal(data, &recs); err != nil {
		log.Printf("[grants] could not parse store: %v", err)
		return
	}
	gs.records = recs
}

// save must be called with gs.mu held.
func (gs *grantStore) save() {
	data, err := json.MarshalIndent(gs.records, "", "  ")
	if err != nil {
		log.Printf("[grants] could not marshal store: %v", err)
		return
	}
	if err := os.WriteFile(gs.path, data, 0o600); err != nil {
		log.Printf("[grants] could not write store %s: %v", gs.path, err)
	}
}

func (gs *grantStore) add(r GrantRecord) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	// Replace any existing record for same IP+port
	filtered := gs.records[:0]
	for _, rec := range gs.records {
		if !(rec.IP == r.IP && rec.Port == r.Port) {
			filtered = append(filtered, rec)
		}
	}
	gs.records = append(filtered, r)
	gs.save()
}

func (gs *grantStore) findByIPPort(ip string, port int) (GrantRecord, bool) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	for _, rec := range gs.records {
		if rec.IP == ip && rec.Port == port {
			return rec, true
		}
	}
	return GrantRecord{}, false
}

func (gs *grantStore) removeByComment(comment string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	filtered := gs.records[:0]
	for _, r := range gs.records {
		if r.Comment != comment {
			filtered = append(filtered, r)
		}
	}
	gs.records = filtered
	gs.save()
}

// revokeGrant removes the iptables NAT rule (if nat mode), removes the UFW
// rule, and drops the record from the persistent store.
func (h *Handlers) revokeGrant(r GrantRecord) {
	portStr := strconv.Itoa(r.Port)
	if r.Mode == "proxy" {
		h.stopBoundProxy(r.ListenHost, r.Port)
	}
	if r.Mode == "nat" {
		stopNATProxy(r.IP, r.Iface, portStr)
	}
	_, _ = runUFW("--force", "delete", "allow", "from", r.IP, "to", "any", "port", portStr, "proto", "tcp")
	_, _ = runUFW("reload")
	h.gs.removeByComment(r.Comment)
	log.Printf("[grants] revoked %s grant for %s:%d", r.Mode, r.IP, r.Port)
}

func proxyKey(host string, port int) string {
	return host + ":" + strconv.Itoa(port)
}

func (h *Handlers) startBoundProxy(host string, port int) error {
	if host == "" {
		return fmt.Errorf("empty listen host")
	}
	key := proxyKey(host, port)
	h.proxyMu.Lock()
	defer h.proxyMu.Unlock()
	if _, ok := h.proxies[key]; ok {
		return nil
	}
	listenAddr := net.JoinHostPort(host, strconv.Itoa(port))
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen on %s failed: %w", listenAddr, err)
	}
	h.proxies[key] = ln
	go func() {
		for {
			client, err := ln.Accept()
			if err != nil {
				return
			}
			go proxyTCP(client, net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		}
	}()
	log.Printf("[proxy] listener started on %s -> 127.0.0.1:%d", listenAddr, port)
	return nil
}

func (h *Handlers) stopBoundProxy(host string, port int) {
	if host == "" {
		return
	}
	key := proxyKey(host, port)
	h.proxyMu.Lock()
	defer h.proxyMu.Unlock()
	if ln, ok := h.proxies[key]; ok {
		_ = ln.Close()
		delete(h.proxies, key)
		log.Printf("[proxy] listener stopped on %s", key)
	}
}

func proxyTCP(client net.Conn, upstreamAddr string) {
	defer client.Close()
	upstream, err := net.DialTimeout("tcp", upstreamAddr, 5*time.Second)
	if err != nil {
		return
	}
	defer upstream.Close()
	done := make(chan struct{}, 2)
	pipe := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go pipe(upstream, client)
	go pipe(client, upstream)
	<-done
}

// ─── iptables NAT proxy ────────────────────────────────────────────────────

// startNATProxy enables route_localnet and inserts PREROUTING + POSTROUTING
// rules so packets from clientIP to public:portStr are DNAT'd to 127.0.0.1 and
// the loopback leg is SNAT'd correctly for replies.
func startNATProxy(clientIP, iface, portStr string) error {
	// Required so the kernel will route DNAT'd packets to the loopback interface.
	if out, err := runSysctl("net.ipv4.conf.all.route_localnet=1"); err != nil {
		return fmt.Errorf("sysctl route_localnet(all): %s: %w", strings.TrimSpace(string(out)), err)
	}
	if iface != "" {
		if out, err := runSysctl("net.ipv4.conf." + iface + ".route_localnet=1"); err != nil {
			return fmt.Errorf("sysctl route_localnet(%s): %s: %w", iface, strings.TrimSpace(string(out)), err)
		}
	}
	out, err := runIPTables("-t", "nat", "-A", "PREROUTING",
		"-i", iface,
		"-p", "tcp", "-s", clientIP, "--dport", portStr,
		"-j", "DNAT", "--to-destination", "127.0.0.1:"+portStr)
	if err != nil {
		return fmt.Errorf("iptables DNAT: %s: %w", strings.TrimSpace(string(out)), err)
	}
	out, err = runIPTables("-t", "nat", "-A", "POSTROUTING",
		"-o", "lo",
		"-p", "tcp", "-s", clientIP, "-d", "127.0.0.1", "--dport", portStr,
		"-j", "MASQUERADE")
	if err != nil {
		_, _ = runIPTables("-t", "nat", "-D", "PREROUTING",
			"-i", iface,
			"-p", "tcp", "-s", clientIP, "--dport", portStr,
			"-j", "DNAT", "--to-destination", "127.0.0.1:"+portStr)
		return fmt.Errorf("iptables MASQUERADE: %s: %w", strings.TrimSpace(string(out)), err)
	}
	log.Printf("[proxy] NAT rules added: iface=%s src=%s port=%s -> 127.0.0.1:%s", iface, clientIP, portStr, portStr)
	return nil
}

func stopNATProxy(clientIP, iface, portStr string) {
	_, _ = runIPTables("-t", "nat", "-D", "PREROUTING",
		"-i", iface,
		"-p", "tcp", "-s", clientIP, "--dport", portStr,
		"-j", "DNAT", "--to-destination", "127.0.0.1:"+portStr)
	_, _ = runIPTables("-t", "nat", "-D", "POSTROUTING",
		"-o", "lo",
		"-p", "tcp", "-s", clientIP, "-d", "127.0.0.1", "--dport", portStr,
		"-j", "MASQUERADE")
	log.Printf("[proxy] NAT rules removed: iface=%s src=%s port=%s", iface, clientIP, portStr)
}

func clearGrantRules(ip, iface, portStr string) {
	for range 8 {
		_, err := runIPTables("-t", "nat", "-D", "PREROUTING",
			"-i", iface,
			"-p", "tcp", "-s", ip, "--dport", portStr,
			"-j", "DNAT", "--to-destination", "127.0.0.1:"+portStr)
		if err != nil {
			break
		}
	}
	for range 8 {
		_, err := runIPTables("-t", "nat", "-D", "POSTROUTING",
			"-o", "lo",
			"-p", "tcp", "-s", ip, "-d", "127.0.0.1", "--dport", portStr,
			"-j", "MASQUERADE")
		if err != nil {
			break
		}
	}
	for range 8 {
		_, err := runUFW("--force", "delete", "allow", "from", ip, "to", "any", "port", portStr, "proto", "tcp")
		if err != nil {
			break
		}
	}
	_, _ = runUFW("reload")
}

func runIPTables(args ...string) ([]byte, error) {
	for _, bin := range []string{"/usr/sbin/iptables", "/sbin/iptables"} {
		if _, err := os.Stat(bin); err == nil {
			return exec.Command("sudo", append([]string{"-n", bin}, args...)...).CombinedOutput()
		}
	}
	return nil, fmt.Errorf("iptables binary not found")
}

func runSysctl(setting string) ([]byte, error) {
	for _, bin := range []string{"/usr/sbin/sysctl", "/sbin/sysctl", "/bin/sysctl"} {
		if _, err := os.Stat(bin); err == nil {
			return exec.Command("sudo", "-n", bin, "-w", setting).CombinedOutput()
		}
	}
	return nil, fmt.Errorf("sysctl binary not found")
}

func detectIngressInterfaceForIP(ip string) string {
	out, err := exec.Command("ip", "route", "get", ip).Output()
	if err != nil {
		return ""
	}
	fields := strings.Fields(string(out))
	for i := 0; i < len(fields)-1; i++ {
		if fields[i] == "dev" {
			return fields[i+1]
		}
	}
	return ""
}

// reconcile revokes all expired records. Safe to call at any time.
func (h *Handlers) reconcileGrants() {
	h.gs.mu.Lock()
	var expired []GrantRecord
	for _, r := range h.gs.records {
		if time.Now().After(r.ExpiresAt) {
			expired = append(expired, r)
		}
	}
	h.gs.mu.Unlock()
	for _, r := range expired {
		h.revokeGrant(r)
	}
}

// InitGrantStore loads the on-disk store, immediately revokes any already-expired
// rules, re-arms timers and restarts NAT rules for still-live grants.
func (h *Handlers) InitGrantStore() {
	h.gs = newGrantStore()
	h.gs.load()

	// Revoke expired immediately (handles grants that outlived a previous restart)
	h.reconcileGrants()

	// Re-arm timers and restart NAT rules for still-live grants
	h.gs.mu.Lock()
	liveCopy := make([]GrantRecord, len(h.gs.records))
	copy(liveCopy, h.gs.records)
	h.gs.mu.Unlock()

	for _, r := range liveCopy {
		r := r
		if r.Mode == "proxy" {
			listenHost := r.ListenHost
			if listenHost == "" {
				listenHost = detectPrimaryIP()
			}
			if err := h.startBoundProxy(listenHost, r.Port); err != nil {
				log.Printf("[grants] could not restart proxy for %s:%d: %v", listenHost, r.Port, err)
			}
		}
		if r.Mode == "nat" {
			portStr := strconv.Itoa(r.Port)
			iface := r.Iface
			if iface == "" {
				iface = detectIngressInterfaceForIP(r.IP)
			}
			if err := startNATProxy(r.IP, iface, portStr); err != nil {
				log.Printf("[grants] could not restart NAT rule for %s:%d: %v", r.IP, r.Port, err)
			}
		}
		remaining := time.Until(r.ExpiresAt)
		if remaining > 0 {
			log.Printf("[grants] re-arming revoke for %s:%d in %s (mode=%s)", r.IP, r.Port, remaining.Round(time.Second), r.Mode)
			time.AfterFunc(remaining, func() { h.revokeGrant(r) })
		}
	}

	// Periodic safety-net reconciler every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			h.reconcileGrants()
		}
	}()

	log.Printf("[grants] store initialised at %s (%d active grants)", h.gs.path, len(liveCopy))
}

// appCatalogEntry maps a process / container name to display metadata and the
// ports we expect it to listen on (used as a fallback when ss is ambiguous).
type appCatalogEntry struct {
	Name     string
	Icon     string
	Color    string
	Category string
	// ConfigFile is read to extract the actual port when non-empty.
	ConfigFile string
	// ConfigPortKey is the key/pattern searched in the config file.
	ConfigPortKey string
	// DefaultPorts lists common ports the app uses (used when config is absent).
	DefaultPorts []int
}

// knownApps is keyed by the process name as it appears in `ss -tlnp` output
// or as a Docker container name prefix.
var knownApps = map[string]appCatalogEntry{
	"grafana": {
		Name: "Grafana", Icon: "mdi-chart-areaspline", Color: "#f46800", Category: "Monitoring",
		ConfigFile: "/etc/grafana/grafana.ini", ConfigPortKey: "http_port",
		DefaultPorts: []int{3000},
	},
	"grafana-server": {
		Name: "Grafana", Icon: "mdi-chart-areaspline", Color: "#f46800", Category: "Monitoring",
		ConfigFile: "/etc/grafana/grafana.ini", ConfigPortKey: "http_port",
		DefaultPorts: []int{3000},
	},
	"prometheus": {
		Name: "Prometheus", Icon: "mdi-database-search", Color: "#e6522c", Category: "Monitoring",
		DefaultPorts: []int{9090},
	},
	"alertmanager": {
		Name: "Alertmanager", Icon: "mdi-bell-alert-outline", Color: "#e6522c", Category: "Monitoring",
		DefaultPorts: []int{9093},
	},
	"node_exporter": {
		Name: "Node Exporter", Icon: "mdi-chart-bar", Color: "#e6522c", Category: "Monitoring",
		DefaultPorts: []int{9100},
	},
	"netdata": {
		Name: "Netdata", Icon: "mdi-chart-timeline", Color: "#00ab44", Category: "Monitoring",
		ConfigFile: "/etc/netdata/netdata.conf", ConfigPortKey: "port",
		DefaultPorts: []int{19999},
	},
	"portainer": {
		Name: "Portainer", Icon: "mdi-docker", Color: "#2496ed", Category: "Containers",
		DefaultPorts: []int{9000, 9443},
	},
	"portainer_ce": {
		Name: "Portainer", Icon: "mdi-docker", Color: "#2496ed", Category: "Containers",
		DefaultPorts: []int{9000, 9443},
	},
	"uptime-kuma": {
		Name: "Uptime Kuma", Icon: "mdi-monitor-heart", Color: "#5cdd8b", Category: "Monitoring",
		DefaultPorts: []int{3001},
	},
	"uptime_kuma": {
		Name: "Uptime Kuma", Icon: "mdi-monitor-heart", Color: "#5cdd8b", Category: "Monitoring",
		DefaultPorts: []int{3001},
	},
	"jenkins": {
		Name: "Jenkins", Icon: "mdi-cog-sync-outline", Color: "#d33833", Category: "CI/CD",
		ConfigFile: "/etc/default/jenkins", ConfigPortKey: "HTTP_PORT",
		DefaultPorts: []int{8080},
	},
	"cadvisor": {
		Name: "cAdvisor", Icon: "mdi-memory", Color: "#2496ed", Category: "Monitoring",
		DefaultPorts: []int{8080},
	},
	"loki": {
		Name: "Loki", Icon: "mdi-text-box-search-outline", Color: "#f9b716", Category: "Logging",
		DefaultPorts: []int{3100},
	},
	"minio": {
		Name: "MinIO", Icon: "mdi-database", Color: "#c72e49", Category: "Storage",
		DefaultPorts: []int{9001},
	},
	"rabbitmq": {
		Name: "RabbitMQ", Icon: "mdi-rabbit", Color: "#ff6600", Category: "Messaging",
		DefaultPorts: []int{15672},
	},
	"elasticsearch": {
		Name: "Elasticsearch", Icon: "mdi-magnify", Color: "#f9b716", Category: "Search",
		DefaultPorts: []int{9200},
	},
	"kibana": {
		Name: "Kibana", Icon: "mdi-chart-pie", Color: "#f9b716", Category: "Logging",
		DefaultPorts: []int{5601},
	},
	"nginx": {
		Name: "Nginx", Icon: "mdi-web", Color: "#009900", Category: "Web Server",
		DefaultPorts: []int{80, 443},
	},
	"apache2": {
		Name: "Apache", Icon: "mdi-web", Color: "#d22128", Category: "Web Server",
		DefaultPorts: []int{80, 443},
	},
	"httpd": {
		Name: "Apache", Icon: "mdi-web", Color: "#d22128", Category: "Web Server",
		DefaultPorts: []int{80, 443},
	},
	"traefik": {
		Name: "Traefik", Icon: "mdi-router-network", Color: "#24a1c1", Category: "Proxy",
		DefaultPorts: []int{8080},
	},
	"cockpit-ws": {
		Name: "Cockpit", Icon: "mdi-gauge", Color: "#0066cc", Category: "Server Management",
		DefaultPorts: []int{9090},
	},
}

// GetTunnelableApps detects web-accessible apps listening on this server and
// returns them so the frontend can generate SSH tunnel commands.
func (h *Handlers) GetTunnelableApps(w http.ResponseWriter, r *http.Request) {
	seen := map[int]bool{} // deduplicate by port
	var apps []TunnelableApp

	// 1. Parse ss -tlnp output
	ssApps := detectFromSS()
	for _, a := range ssApps {
		if !seen[a.Port] {
			seen[a.Port] = true
			apps = append(apps, a)
		}
	}

	// 2. Docker containers with exposed ports
	dockerApps := detectFromDocker()
	for _, a := range dockerApps {
		if !seen[a.Port] {
			seen[a.Port] = true
			apps = append(apps, a)
		}
	}

	// Sort by port
	sort.Slice(apps, func(i, j int) bool { return apps[i].Port < apps[j].Port })

	host := detectServerHost(r)
	sshPort := detectSSHPort()
	sshUser, userSource := detectPreferredSSHUser()
	if sshPort <= 0 {
		sshPort = 22
	}

	writeJSON(w, http.StatusOK, TunnelableAppsResponse{
		Apps: apps,
		Connection: TunnelConnectionInfo{
			Host:          host,
			SSHPort:       sshPort,
			SSHUser:       sshUser,
			SSHUserSource: userSource,
		},
	})
}

// GrantTunnelAccess temporarily allows the caller IP to access one app port.
// For localhost-bound ports (e.g. Docker with 127.0.0.1 binding) it starts
// an in-process TCP proxy on 0.0.0.0:PORT → 127.0.0.1:PORT. For ports already
// on 0.0.0.0 it falls back to a UFW rule only.
func (h *Handlers) GrantTunnelAccess(w http.ResponseWriter, r *http.Request) {
	var req grantTunnelAccessRequest
	if err := jsonDecodeBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Port < 1 || req.Port > 65535 {
		writeError(w, http.StatusBadRequest, "invalid port")
		return
	}

	ip := strings.TrimSpace(clientIP(r))
	if net.ParseIP(ip) == nil {
		writeError(w, http.StatusBadRequest, "could not determine a valid client IP")
		return
	}

	// Validate duration: allowed 1, 3, 6, 12, 24 hours; default 3
	allowedDurations := map[int]bool{1: true, 3: true, 6: true, 12: true, 24: true}
	durHours := req.DurationHours
	if !allowedDurations[durHours] {
		durHours = 3
	}

	portStr := strconv.Itoa(req.Port)
	grantFor := time.Duration(durHours) * time.Hour
	expiresAt := time.Now().Add(grantFor)
	comment := fmt.Sprintf("sc-temp-%d-%d", req.Port, time.Now().Unix())

	iface := detectIngressInterfaceForIP(ip)
	listenHost := detectPrimaryIP()
	if existing, ok := h.gs.findByIPPort(ip, req.Port); ok {
		h.revokeGrant(existing)
	}
	clearGrantRules(ip, iface, portStr)

	// Determine mode: if the port is localhost-only (Docker with 127.0.0.1 binding)
	// use iptables DNAT so packets from clientIP are redirected to 127.0.0.1:port
	// at the kernel level — no socket binding conflict. Otherwise UFW is sufficient.
	mode := "ufw"
	if isLocalhostBound(req.Port) {
		if err := h.startBoundProxy(listenHost, req.Port); err != nil {
			writeError(w, http.StatusInternalServerError,
				fmt.Sprintf("port %d is localhost-only; proxy setup failed: %s", req.Port, err))
			return
		}
		mode = "proxy"
	}

	// UFW rule: provides IP-restriction in both modes.
	ufwOk := false
	if sudoAvailable() {
		if out, err := runUFW("allow", "from", ip, "to", "any", "port", portStr, "proto", "tcp", "comment", comment); err != nil {
			errMsg := strings.TrimSpace(string(out))
			if errMsg == "" {
				errMsg = err.Error()
			}
			if mode == "ufw" {
				writeError(w, http.StatusInternalServerError, "failed to grant access: "+errMsg)
				return
			}
			// non-fatal for proxy mode — listener is already bound and UFW may not be governing the path
			log.Printf("[grants] UFW add failed (non-fatal in %s mode): %s", mode, errMsg)
		} else {
			_, _ = runUFW("reload")
			ufwOk = true
		}
	}

	if mode == "ufw" && !ufwOk {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	// Persist and arm expiry timer
	rec := GrantRecord{IP: ip, Port: req.Port, Comment: comment, ExpiresAt: expiresAt, Mode: mode, Iface: iface, ListenHost: listenHost}
	h.gs.add(rec)
	time.AfterFunc(grantFor, func() { h.revokeGrant(rec) })

	writeJSON(w, http.StatusOK, map[string]any{
		"status":         "granted",
		"ip":             ip,
		"port":           req.Port,
		"mode":           mode,
		"duration_hours": durHours,
		"duration_sec":   int(grantFor.Seconds()),
		"expires_at":     expiresAt.Unix(),
	})
}

func jsonDecodeBody(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// isLocalhostBound returns true if any process is listening on 127.0.0.1:port
// (but NOT on 0.0.0.0:port), meaning the service is localhost-only and needs
// the in-process proxy to be reachable from outside.
func isLocalhostBound(port int) bool {
	portStr := strconv.Itoa(port)
	out, err := exec.Command("ss", "-tlnp").Output()
	if err != nil {
		return false
	}
	hasLocalhost := false
	hasPublic := false
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "LISTEN") {
			continue
		}
		// Match address:port in the local-address column (4th field)
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		addr := fields[3] // e.g. "127.0.0.1:9000" or "0.0.0.0:9000" or "*:9000"
		_, addrPort, err := net.SplitHostPort(addr)
		if err != nil || addrPort != portStr {
			continue
		}
		host, _, _ := net.SplitHostPort(addr)
		ip := net.ParseIP(host)
		if ip != nil && ip.IsLoopback() {
			hasLocalhost = true
		} else {
			// 0.0.0.0 or :: — already public
			hasPublic = true
		}
	}
	return hasLocalhost && !hasPublic
}

// detectFromSS parses `ss -tlnp` and matches process names against knownApps.
func detectFromSS() []TunnelableApp {
	out, err := exec.Command("ss", "-tlnp").Output()
	if err != nil {
		return nil
	}

	// Example line:
	// LISTEN 0 128 0.0.0.0:9090 0.0.0.0:* users:(("prometheus",pid=1234,fd=6))
	rePort := regexp.MustCompile(`:(\d+)\s`)
	reProc := regexp.MustCompile(`"([^"]+)"`)

	var results []TunnelableApp
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "LISTEN") {
			continue
		}

		portM := rePort.FindStringSubmatch(line)
		if portM == nil {
			continue
		}
		port, err := strconv.Atoi(portM[1])
		if err != nil || port <= 0 || port > 65535 {
			continue
		}
		// Skip SentinelCore's own port to avoid self-reference
		if port == 8080 || port == 8443 {
			continue
		}

		procM := reProc.FindStringSubmatch(line)
		if procM == nil {
			continue
		}
		procName := strings.ToLower(procM[1])

		entry, ok := matchCatalog(procName)
		if !ok {
			continue
		}

		// Try to read actual port from config file
		configPort := readPortFromConfig(entry)
		if configPort > 0 {
			port = configPort
		}

		results = append(results, TunnelableApp{
			Name:        entry.Name,
			Port:        port,
			Icon:        entry.Icon,
			Color:       entry.Color,
			Category:    entry.Category,
			Source:      "process",
			ProcessName: procName,
		})
	}
	return results
}

// detectFromDocker reads running Docker containers and their exposed host ports.
func detectFromDocker() []TunnelableApp {
	out, err := exec.Command("docker", "ps", "--format", "{{.Names}}\t{{.Ports}}").Output()
	if err != nil {
		return nil
	}

	// Ports field examples:
	// 0.0.0.0:9000->9000/tcp, 0.0.0.0:9443->9443/tcp
	// :::3000->3000/tcp
	reHostPort := regexp.MustCompile(`(?:0\.0\.0\.0|::):(\d+)->`)

	var results []TunnelableApp
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		containerName := strings.ToLower(parts[0])
		portField := parts[1]

		entry, ok := matchCatalog(containerName)
		if !ok {
			continue
		}

		matches := reHostPort.FindAllStringSubmatch(portField, -1)
		added := map[int]bool{}
		for _, m := range matches {
			port, err := strconv.Atoi(m[1])
			if err != nil || added[port] {
				continue
			}
			if port == 8080 || port == 8443 {
				continue
			}
			added[port] = true
			results = append(results, TunnelableApp{
				Name:        entry.Name,
				Port:        port,
				Icon:        entry.Icon,
				Color:       entry.Color,
				Category:    entry.Category,
				Source:      "docker",
				ProcessName: containerName,
			})
		}
	}
	return results
}

// matchCatalog does a fuzzy match of a process/container name against knownApps.
func matchCatalog(name string) (appCatalogEntry, bool) {
	// Exact match first
	if e, ok := knownApps[name]; ok {
		return e, true
	}
	// Prefix / contains match
	for key, entry := range knownApps {
		if strings.Contains(name, key) || strings.Contains(key, name) {
			return entry, true
		}
	}
	return appCatalogEntry{}, false
}

// readPortFromConfig reads a simple key=value or key: value config file and
// extracts the numeric port for the given key.
func readPortFromConfig(entry appCatalogEntry) int {
	if entry.ConfigFile == "" || entry.ConfigPortKey == "" {
		return 0
	}
	f, err := os.Open(entry.ConfigFile)
	if err != nil {
		return 0
	}
	defer f.Close()

	key := strings.ToLower(entry.ConfigPortKey)
	re := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*[=:]\s*(\d+)`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.ToLower(strings.TrimSpace(scanner.Text()))
		// Skip comments
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		m := re.FindStringSubmatch(line)
		if m != nil {
			port, err := strconv.Atoi(m[1])
			if err == nil && port > 0 {
				return port
			}
		}
	}
	return 0
}

func detectServerHost(r *http.Request) string {
	if ip := detectPrimaryIP(); ip != "" {
		return ip
	}
	host := r.Host
	if strings.Contains(host, ":") {
		h, _, err := net.SplitHostPort(host)
		if err == nil && h != "" {
			return h
		}
	}
	if host != "" {
		return host
	}
	return "127.0.0.1"
}

func detectPrimaryIP() string {
	conn, err := net.Dial("udp", "1.1.1.1:53")
	if err != nil {
		return ""
	}
	defer conn.Close()
	if addr, ok := conn.LocalAddr().(*net.UDPAddr); ok && addr.IP != nil {
		if v4 := addr.IP.To4(); v4 != nil {
			return v4.String()
		}
		return addr.IP.String()
	}
	return ""
}

func detectSSHPort() int {
	if p := detectSSHPortFromSS(); p > 0 {
		return p
	}
	if p := detectSSHPortFromConfig(); p > 0 {
		return p
	}
	return 22
}

func detectSSHPortFromSS() int {
	out, err := exec.Command("ss", "-tlnp").Output()
	if err != nil {
		return 0
	}
	rePort := regexp.MustCompile(`:(\d+)\s`)
	reProc := regexp.MustCompile(`"([^"]+)"`)

	ports := map[int]bool{}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "LISTEN") {
			continue
		}
		proc := reProc.FindStringSubmatch(line)
		if proc == nil || !strings.Contains(strings.ToLower(proc[1]), "sshd") {
			continue
		}
		pm := rePort.FindStringSubmatch(line)
		if pm == nil {
			continue
		}
		p, err := strconv.Atoi(pm[1])
		if err == nil && p > 0 {
			ports[p] = true
		}
	}
	if len(ports) == 0 {
		return 0
	}
	minPort := 65535
	for p := range ports {
		if p < minPort {
			minPort = p
		}
	}
	return minPort
}

func detectSSHPortFromConfig() int {
	files := []string{"/etc/ssh/sshd_config"}
	if matches, err := filepath.Glob("/etc/ssh/sshd_config.d/*.conf"); err == nil && len(matches) > 0 {
		files = append(files, matches...)
	}

	rePort := regexp.MustCompile(`(?i)^\s*port\s+(\d+)\s*$`)
	for _, f := range files {
		fd, err := os.Open(f)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(fd)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			m := rePort.FindStringSubmatch(line)
			if m != nil {
				p, err := strconv.Atoi(m[1])
				fd.Close()
				if err == nil && p > 0 {
					return p
				}
			}
		}
		fd.Close()
	}
	return 0
}

func detectPreferredSSHUser() (string, string) {
	if userExists("deploy") {
		return "deploy", "deploy"
	}
	out, err := exec.Command("systemctl", "show", "sentinelcore", "--property=User").Output()
	if err == nil {
		v := strings.TrimSpace(string(out))
		if strings.HasPrefix(v, "User=") {
			u := strings.TrimSpace(strings.TrimPrefix(v, "User="))
			if u != "" {
				return u, "service"
			}
		}
	}
	if u, err := user.Current(); err == nil && u != nil && u.Username != "" {
		name := u.Username
		if strings.Contains(name, "\\") {
			parts := strings.Split(name, "\\")
			name = parts[len(parts)-1]
		}
		return name, "current"
	}
	return "root", "fallback"
}

func userExists(username string) bool {
	fd, err := os.Open("/etc/passwd")
	if err != nil {
		return false
	}
	defer fd.Close()
	prefix := username + ":"
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			parts := strings.Split(line, ":")
			if len(parts) >= 7 {
				shell := parts[6]
				if strings.Contains(shell, "nologin") || strings.Contains(shell, "false") {
					return false
				}
			}
			return true
		}
	}
	return false
}

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ehsanR91/sentinelcore/internal/db"
	"github.com/ehsanR91/sentinelcore/internal/monitoring"
)

type managedService struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Package     string `json:"package"`
	Config      string `json:"config"`
	Unit        string `json:"unit,omitempty"`
	Running     bool   `json:"running"`
	Installed   bool   `json:"installed"`
	ActiveState string `json:"active_state"`
	SubState    string `json:"sub_state"`
	Status      string `json:"status"`
}

// Service installation state for tracking background installations
var (
	svcInstallLogs    []string
	svcInstallDone    bool
	svcInstallErr     string
	svcInstallRunning bool
	svcInstallMu      sync.Mutex

	managedServicesCacheMu     sync.Mutex
	managedServicesCache       = map[string]managedServiceCacheEntry{}
	managedServicesRefreshWake = make(chan struct{}, 1)
	managedServicesRefreshOnce sync.Once
)

const (
	managedServicesCacheTTL      = 10 * time.Second
	managedServicesRefreshPeriod = 20 * time.Second
)

type managedServiceCacheEntry struct {
	expiresAt time.Time
	services  []managedService
}

type serviceCatalogItem struct {
	Label   string
	Package string
	Config  string
	// RequireBinary can be used for tools where a loaded systemd unit can remain
	// even if the package/binary is removed (or where the unit state is misleading).
	// When set, "installed" requires that the binary exists in PATH.
	RequireBinary string
	// Unit is the systemd unit name to query for status. Defaults to the
	// catalog key when empty. Needed when the unit name differs from the
	// package/key name (e.g. sshd → ssh on Debian/Ubuntu).
	Unit string
	// AltUnits lists additional unit names tried when Unit returns not-found.
	// Handles distro differences (sshd on RHEL, ssh on Debian).
	AltUnits []string
	// CronBased marks tools that have no persistent daemon — they run via
	// cron/systemd timer. "Installed" is checked via the package manager;
	// "Running" checks the associated timer unit instead.
	CronBased bool
	// TimerUnit is the systemd timer or oneshot unit to check for cron-based
	// tools (e.g. dailyaidecheck.timer for aide).
	TimerUnit string
}

var serviceCatalog = map[string]serviceCatalogItem{
	"ufw": {
		Label: "UFW", Package: "ufw", Config: "", RequireBinary: "ufw",
	},
	"fail2ban": {
		Label: "fail2ban", Package: "fail2ban", Config: "/etc/fail2ban/jail.local",
	},
	"crowdsec": {
		Label: "CrowdSec", Package: "crowdsec", Config: "/etc/crowdsec/config.yaml",
	},
	"psad": {
		Label: "psad", Package: "psad", Config: "/etc/psad/psad.conf",
	},
	"clamav-daemon": {
		Label: "ClamAV", Package: "clamav clamav-daemon", Config: "/etc/clamav/clamd.conf",
		AltUnits: []string{"clamd"},
	},
	"auditd": {
		Label: "auditd", Package: "auditd", Config: "/etc/audit/auditd.conf",
	},
	"apparmor": {
		Label: "AppArmor", Package: "apparmor", Config: "/etc/apparmor/parser.conf",
	},
	"docker": {
		Label: "Docker", Package: "docker.io", Config: "/etc/docker/daemon.json",
		AltUnits: []string{"docker.service"},
	},
	"netdata": {
		Label: "Netdata", Package: "netdata", Config: "/etc/netdata/netdata.conf",
	},
	"unattended-upgrades": {
		Label: "Auto-Update", Package: "unattended-upgrades",
		Config:    "/etc/apt/apt.conf.d/50unattended-upgrades",
		CronBased: true,
		TimerUnit: "apt-daily-upgrade.timer",
	},
	"aide": {
		Label: "AIDE", Package: "aide", Config: "/etc/aide/aide.conf",
		CronBased: true,
		TimerUnit: "dailyaidecheck.timer",
	},
	"rkhunter": {
		Label: "rkhunter", Package: "rkhunter", Config: "/etc/rkhunter.conf",
		CronBased: true,
	},
	"nginx": {
		Label: "nginx", Package: "nginx", Config: "/etc/nginx/nginx.conf",
	},
	// sshd: Debian/Ubuntu use "ssh"; RHEL/Fedora use "sshd". AltUnits handles both.
	"sshd": {
		Label: "sshd", Package: "openssh-server", Config: "/etc/ssh/sshd_config",
		Unit: "ssh", AltUnits: []string{"sshd"},
	},
}

var allowedServiceConfigKeys = map[string]bool{
	"psad_email_alert_danger_level": true,
	"psad_enable_auto_ids":          true,
	"psad_host_deny_proto":          true,
	"clamav_max_file_size":          true,
	"clamav_scan_archive":           true,
	"aide_check_frequency_hours":    true,
	"rkhunter_update_before_scan":   true,
	"rkhunter_mail_on_warning":      true,
	"fail2ban_bantime_seconds":      true,
	"fail2ban_findtime_seconds":     true,
	"fail2ban_maxretry":             true,
}

func (h *Handlers) GetManagedServices(w http.ResponseWriter, r *http.Request) {
	keys := make([]string, 0, len(serviceCatalog))
	for name := range serviceCatalog {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	writeJSON(w, http.StatusOK, buildManagedServices(keys))
}

func (h *Handlers) StartManagedServicesRefresher() {
	managedServicesRefreshOnce.Do(func() {
		go h.runManagedServicesRefresher()
	})
	wakeManagedServicesRefresher()
}

func invalidateManagedServicesCache() {
	managedServicesCacheMu.Lock()
	managedServicesCache = map[string]managedServiceCacheEntry{}
	managedServicesCacheMu.Unlock()
	wakeManagedServicesRefresher()
}

func cloneManagedServices(items []managedService) []managedService {
	return append([]managedService(nil), items...)
}

func wakeManagedServicesRefresher() {
	select {
	case managedServicesRefreshWake <- struct{}{}:
	default:
	}
}

func managedServicesCacheIdentity(keys []string) ([]string, string) {
	cacheKeys := append([]string(nil), keys...)
	sort.Strings(cacheKeys)
	return cacheKeys, strings.Join(cacheKeys, "|")
}

func storeManagedServicesCache(cacheKey string, services []managedService, ttl time.Duration) {
	managedServicesCacheMu.Lock()
	managedServicesCache[cacheKey] = managedServiceCacheEntry{
		expiresAt: time.Now().Add(ttl),
		services:  cloneManagedServices(services),
	}
	managedServicesCacheMu.Unlock()
}

func buildManagedServices(keys []string) []managedService {
	_, cacheKey := managedServicesCacheIdentity(keys)

	managedServicesCacheMu.Lock()
	if entry, ok := managedServicesCache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		cached := cloneManagedServices(entry.services)
		managedServicesCacheMu.Unlock()
		return cached
	}
	managedServicesCacheMu.Unlock()
	return buildManagedServicesFresh(keys)
}

func buildManagedServicesFresh(keys []string) []managedService {
	_, cacheKey := managedServicesCacheIdentity(keys)

	unitNames := make([]string, 0, len(keys)*3)
	for _, name := range keys {
		meta := serviceCatalog[name]
		primary := name
		if meta.Unit != "" {
			primary = meta.Unit
		}
		unitNames = append(unitNames, primary)
		unitNames = append(unitNames, meta.AltUnits...)
		if meta.TimerUnit != "" {
			unitNames = append(unitNames, meta.TimerUnit)
		}
	}

	statuses := monitoring.CheckServices(unitNames)
	byUnit := map[string]monitoring.ServiceStatus{}
	for _, st := range statuses {
		byUnit[st.Name] = st
	}

	out := make([]managedService, 0, len(keys))
	for _, name := range keys {
		meta := serviceCatalog[name]
		installed := serviceInstalled(name, meta)
		st := resolveServiceStatus(name, meta, byUnit)

		if meta.CronBased && installed && (st.ActiveState == "" || st.ActiveState == "inactive") {
			st.ActiveState = "active"
			st.SubState = "installed"
		}

		running := st.IsRunning
		if meta.CronBased && installed {
			running = st.ActiveState == "active" || installed
		}
		if !installed {
			running = false
		}

		unitName := st.Name
		if unitName == "" {
			switch {
			case meta.Unit != "":
				unitName = meta.Unit
			case meta.TimerUnit != "":
				unitName = meta.TimerUnit
			default:
				unitName = name
			}
		}

		out = append(out, managedService{
			Name:        name,
			Label:       meta.Label,
			Package:     meta.Package,
			Config:      meta.Config,
			Unit:        unitName,
			Running:     running,
			Installed:   installed,
			ActiveState: st.ActiveState,
			SubState:    st.SubState,
			Status:      managedServiceStatus(installed, running, st),
		})
	}

	storeManagedServicesCache(cacheKey, out, managedServicesCacheTTL)
	return out
}

func (h *Handlers) runManagedServicesRefresher() {
	h.refreshManagedServiceSnapshots()
	ticker := time.NewTicker(managedServicesRefreshPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
		case <-managedServicesRefreshWake:
		}
		h.refreshManagedServiceSnapshots()
	}
}

func (h *Handlers) refreshManagedServiceSnapshots() {
	for _, keys := range h.managedServicesPrewarmSets() {
		buildManagedServicesFresh(keys)
	}
}

func (h *Handlers) managedServicesPrewarmSets() [][]string {
	all := make([]string, 0, len(serviceCatalog))
	for name := range serviceCatalog {
		all = append(all, name)
	}
	security := []string{"ufw", "fail2ban", "crowdsec", "psad", "clamav-daemon", "auditd", "apparmor", "docker", "aide", "rkhunter"}
	return [][]string{
		append([]string(nil), h.cfg.WatchedServices...),
		all,
		security,
	}
}

func managedServiceStatus(installed, running bool, st monitoring.ServiceStatus) string {
	if !installed {
		return "missing"
	}
	if st.ActiveState == "failed" || st.SubState == "failed" {
		return "failed"
	}
	if running {
		return "active"
	}
	if st.ActiveState == "inactive" || st.ActiveState == "" {
		return "inactive"
	}
	return st.ActiveState
}

func resolveServiceStatus(name string, meta serviceCatalogItem, byUnit map[string]monitoring.ServiceStatus) monitoring.ServiceStatus {
	candidates := []string{name}
	if meta.Unit != "" {
		candidates = []string{meta.Unit}
	}
	candidates = append(candidates, meta.AltUnits...)
	if meta.TimerUnit != "" {
		candidates = append(candidates, meta.TimerUnit)
	}

	for _, u := range candidates {
		if st, ok := byUnit[u]; ok && st.ActiveState != "inactive" && st.ActiveState != "" {
			return st
		}
	}
	for _, u := range candidates {
		if st, ok := byUnit[u]; ok {
			return st
		}
	}
	return monitoring.ServiceStatus{Name: name, Label: meta.Label, ActiveState: "inactive", SubState: "dead"}
}

func binaryExists(name string) bool {
	if name == "" {
		return true
	}
	_, err := exec.LookPath(name)
	return err == nil
}

// lookCmd returns the absolute path for a command, searching known sbin/bin locations
// before falling back to exec.LookPath. Avoids PATH-dependent failures in systemd units.
func lookCmd(name string) string {
	for _, prefix := range []string{"/usr/sbin", "/usr/bin", "/sbin", "/bin", "/usr/local/sbin", "/usr/local/bin"} {
		p := prefix + "/" + name
		if info, err := os.Stat(p); err == nil && info.Mode()&0o111 != 0 {
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return name // fall back to name and let the OS resolve it
}

func (h *Handlers) ServiceInstall(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := serviceCatalog[name]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported service")
		return
	}
	if serviceInstalled(name, meta) {
		writeJSON(w, http.StatusOK, map[string]any{"status": "already_installed"})
		return
	}

	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	// Guard against concurrent installs — only one at a time.
	svcInstallMu.Lock()
	if svcInstallRunning {
		svcInstallMu.Unlock()
		writeError(w, http.StatusConflict, "another installation is already in progress")
		return
	}
	svcInstallLogs = []string{}
	svcInstallDone = false
	svcInstallErr = ""
	svcInstallRunning = true
	svcInstallMu.Unlock()

	go func() {
		defer func() {
			invalidateManagedServicesCache()
			svcInstallMu.Lock()
			svcInstallRunning = false
			svcInstallMu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		// Log start
		svcInstallMu.Lock()
		svcInstallLogs = append(svcInstallLogs, fmt.Sprintf("Starting installation of %s...", name))
		svcInstallMu.Unlock()

		// Fix any repos with missing GPG keys (e.g. netdata packagecloud repo
		// left unconfigured by kickstart on Ubuntu 24.04) before the real update.
		if fixed := fixMissingRepoKeys(ctx); len(fixed) > 0 {
			svcInstallMu.Lock()
			svcInstallLogs = append(svcInstallLogs, fmt.Sprintf("Fixed missing GPG keys: %s", strings.Join(fixed, ", ")))
			svcInstallMu.Unlock()
		}

		// Run apt-get update and capture output.
		// APT::Update::Error-Mode=any makes individual repo failures (e.g. missing GPG key)
		// non-fatal so a single broken repo doesn't abort the whole update.
		out, err := runPrivilegedEnv(ctx, "apt-get", "-y", "-o", "APT::Update::Error-Mode=any", "update")
		svcInstallMu.Lock()
		for _, line := range strings.Split(string(out), "\n") {
			if line != "" {
				svcInstallLogs = append(svcInstallLogs, line)
			}
		}
		if err != nil {
			// Log as a warning but continue — packages on working repos are still installable.
			svcInstallLogs = append(svcInstallLogs, fmt.Sprintf("Warning: apt update had errors (%v), continuing installation...", err))
		}
		svcInstallMu.Unlock()

		// Run apt-get install and capture output
		pkgParts := strings.Fields(meta.Package)
		installArgs := append([]string{"-y", "install"}, pkgParts...)
		out, err = runPrivilegedEnv(ctx, "apt-get", installArgs...)
		if err != nil {
			svcInstallMu.Lock()
			svcInstallLogs = append(svcInstallLogs, fmt.Sprintf("Error during installation: %v", err))
			svcInstallErr = err.Error()
			svcInstallDone = true
			svcInstallMu.Unlock()
			return
		}

		// Sanity check: some services may appear to install but not provide the expected binary.
		if meta.RequireBinary != "" && !binaryExists(meta.RequireBinary) {
			svcInstallMu.Lock()
			svcInstallLogs = append(svcInstallLogs, fmt.Sprintf("Error: expected binary '%s' was not found in PATH after install", meta.RequireBinary))
			svcInstallErr = "binary not found after install"
			svcInstallDone = true
			svcInstallMu.Unlock()
			return
		}
		svcInstallMu.Lock()
		for _, line := range strings.Split(string(out), "\n") {
			if line != "" {
				svcInstallLogs = append(svcInstallLogs, line)
			}
		}
		svcInstallMu.Unlock()

		// Enable and start service (only if systemd unit exists)
		unit := name
		if meta.Unit != "" {
			unit = meta.Unit
		}
		candidates := append([]string{unit}, meta.AltUnits...)
		started := false
		for _, u := range candidates {
			if !unitLoaded(u) {
				continue
			}
			_, err := runPrivileged(context.Background(), "systemctl", "enable", "--now", u)
			svcInstallMu.Lock()
			if err != nil {
				svcInstallLogs = append(svcInstallLogs, fmt.Sprintf("Warning: systemctl enable --now %s failed: %v", u, err))
				svcInstallMu.Unlock()
				continue
			}
			svcInstallLogs = append(svcInstallLogs, fmt.Sprintf("Service %s enabled and started", u))
			svcInstallMu.Unlock()
			started = true
			break
		}
		if !started {
			svcInstallMu.Lock()
			svcInstallLogs = append(svcInstallLogs, "Note: no systemd unit found to enable/start (package may be installed but has no persistent service)")
			svcInstallMu.Unlock()
		}

		svcInstallMu.Lock()
		svcInstallDone = true
		svcInstallMu.Unlock()

		_ = db.InsertAlert("service", "info", "services", fmt.Sprintf("Installed %s", name), "", "system")
	}()

	writeJSON(w, http.StatusOK, map[string]any{"status": "installing", "message": "Installation running in background"})
}

// GetServiceInstallLogs returns the current service installation logs and completion status
func (h *Handlers) GetServiceInstallLogs(w http.ResponseWriter, r *http.Request) {
	svcInstallMu.Lock()
	defer svcInstallMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"logs":  svcInstallLogs,
		"done":  svcInstallDone,
		"error": svcInstallErr,
	})
}

func (h *Handlers) ServiceAction(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	action := chi.URLParam(r, "action")
	meta, ok := serviceCatalog[name]
	if !ok {
		writeError(w, http.StatusBadRequest, "unsupported service")
		return
	}
	allowed := map[string]bool{
		"start":     true,
		"stop":      true,
		"restart":   true,
		"enable":    true,
		"disable":   true,
		"reinstall": true,
		"uninstall": true,
	}
	if !allowed[action] {
		writeError(w, http.StatusBadRequest, "unsupported action")
		return
	}
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 90*time.Second)
	defer cancel()

	pkgParts := strings.Fields(meta.Package)
	runApt := func(args ...string) (string, error) {
		return runPrivilegedEnv(ctx, "apt-get", args...)
	}

	if action == "reinstall" {
		if _, err := runApt(append([]string{"-y", "--reinstall", "install"}, pkgParts...)...); err != nil {
			writeError(w, http.StatusBadGateway, "service reinstall failed: "+err.Error())
			return
		}
		if meta.RequireBinary != "" && !binaryExists(meta.RequireBinary) {
			writeError(w, http.StatusBadGateway, "service reinstall succeeded but required binary is missing")
			return
		}
		invalidateManagedServicesCache()
		_ = db.InsertAlert("service", "info", "services", fmt.Sprintf("reinstalled %s", name), "", "system")
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "action": action, "service": name})
		return
	}

	if action == "uninstall" {
		unitNames := []string{name}
		if meta.Unit != "" {
			unitNames = append([]string{meta.Unit}, unitNames...)
		}
		unitNames = append(unitNames, meta.AltUnits...)
		for _, u := range unitNames {
			if unitLoaded(u) {
				_, _ = runPrivileged(ctx, "systemctl", "disable", "--now", u)
			}
		}
		if _, err := runApt(append([]string{"-y", "remove", "--purge"}, pkgParts...)...); err != nil {
			writeError(w, http.StatusBadGateway, "service uninstall failed: "+err.Error())
			return
		}
		invalidateManagedServicesCache()
		_ = db.InsertAlert("service", "info", "services", fmt.Sprintf("uninstalled %s", name), "", "system")
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "action": action, "service": name})
		return
	}

	// Resolve to an existing unit name before executing actions.
	// Avoid "zombie" behavior where systemctl reports about a unit that isn't loaded.
	unit := name
	if meta.Unit != "" {
		unit = meta.Unit
	}
	unitCandidates := append([]string{unit}, meta.AltUnits...)
	resolved := ""
	for _, u := range unitCandidates {
		if unitLoaded(u) {
			resolved = u
			break
		}
	}
	if resolved == "" {
		pkgName := strings.Fields(meta.Package)[0]
		hint := fmt.Sprintf("Make sure %s is installed and the unit exists. On the server run: sudo apt-get install %s && sudo systemctl daemon-reload && sudo systemctl start %s", pkgName, pkgName, name)
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"error": "systemd unit not found for service",
			"hint":  hint,
		})
		return
	}

	if _, err := runPrivileged(ctx, "systemctl", action, resolved); err != nil {
		writeError(w, http.StatusBadGateway, "service action failed: "+err.Error())
		return
	}
	invalidateManagedServicesCache()
	_ = db.InsertAlert("service", "info", "services", fmt.Sprintf("%s %s", action, name), "", "system")
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "action": action, "service": name, "unit": resolved})
}

func (h *Handlers) GetServiceConfig(w http.ResponseWriter, r *http.Request) {
	out := map[string]string{}
	for key := range allowedServiceConfigKeys {
		out[key] = db.GetSetting("svc_cfg_"+key, "")
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) UpdateServiceConfig(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	for key, val := range body {
		if !allowedServiceConfigKeys[key] {
			continue
		}
		_ = db.SetSetting("svc_cfg_"+key, strings.TrimSpace(val))
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// unitLoaded returns true if systemctl knows the given unit name.
func unitLoaded(unit string) bool {
	out, err := exec.Command("systemctl", "show", "--property=LoadState", unit).Output()
	return err == nil && !strings.Contains(string(out), "LoadState=not-found")
}

// packageInstalled checks whether a package is installed via the native package manager.
func packageInstalled(pkg string) bool {
	// apt / dpkg  (Debian, Ubuntu, Mint, …)
	if out, err := exec.Command("dpkg-query", "-W", "-f=${Status}", pkg).Output(); err == nil {
		if strings.Contains(string(out), "install ok installed") {
			return true
		}
	}
	// rpm  (RHEL, Fedora, Rocky, AlmaLinux, CentOS, …)
	if exec.Command("rpm", "-q", "--quiet", pkg).Run() == nil {
		return true
	}
	// pacman  (Arch, Manjaro, …)
	if exec.Command("pacman", "-Q", pkg).Run() == nil {
		return true
	}
	// zypper / rpm-based openSUSE — covered by the rpm check above
	return false
}

// serviceInstalled returns true when the service package is present on the system.
// It checks the primary systemd unit first (fast), then falls back to querying
// the package manager directly — needed for cron-based tools (aide, rkhunter)
// and for services whose unit name differs across distros.
func serviceInstalled(name string, meta serviceCatalogItem) bool {
	if meta.RequireBinary != "" && !binaryExists(meta.RequireBinary) {
		return false
	}

	// Determine which unit names to probe
	units := []string{name}
	if meta.Unit != "" {
		units = []string{meta.Unit}
	}
	units = append(units, meta.AltUnits...)
	if meta.TimerUnit != "" {
		units = append(units, meta.TimerUnit)
	}

	// Fast path: any known unit is loaded → package is definitely present
	for _, u := range units {
		if unitLoaded(u) {
			return true
		}
	}

	// Slow path: query the package manager for the first package token
	pkg := strings.Fields(meta.Package)[0]
	return packageInstalled(pkg)
}

type schedulerState struct {
	mu      sync.Mutex
	running map[int64]bool
	wake    chan struct{}
}

var taskScheduler = schedulerState{
	running: map[int64]bool{},
	wake:    make(chan struct{}, 1),
}

func notifyTaskScheduler() {
	select {
	case taskScheduler.wake <- struct{}{}:
	default:
	}
}

func (h *Handlers) StartTaskScheduler() {
	go func() {
		for {
			delay := h.runDueTasks()
			if delay < time.Second {
				delay = time.Second
			}

			timer := time.NewTimer(delay)
			select {
			case <-timer.C:
			case <-taskScheduler.wake:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
			}
		}
	}()
}

func (h *Handlers) runDueTasks() time.Duration {
	tasks, err := db.ListTasks()
	if err != nil {
		return time.Minute
	}
	if len(tasks) == 0 {
		return 5 * time.Minute
	}

	taskIDs := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	latestRuns, err := db.LatestRunsByTaskIDs(taskIDs)
	if err != nil {
		return time.Minute
	}

	now := time.Now()
	nextDelay := 5 * time.Minute
	hasScheduledTask := false
	for _, t := range tasks {
		if !t.Enabled || t.ScheduleKind != "interval" {
			continue
		}
		intervalSec, err := strconv.Atoi(strings.TrimSpace(t.ScheduleExpr))
		if err != nil || intervalSec < 30 {
			continue
		}
		hasScheduledTask = true
		interval := time.Duration(intervalSec) * time.Second
		last := latestRuns[t.ID]
		if last != nil && last.Status == "running" {
			if 10*time.Second < nextDelay {
				nextDelay = 10 * time.Second
			}
			continue
		}

		dueAt := now
		if last != nil {
			dueAt = time.Unix(last.StartedAt, 0).Add(interval)
		}
		if !dueAt.After(now) {
			if h.runTaskAsync(t, "scheduler") {
				if interval < nextDelay {
					nextDelay = interval
				}
			} else if 10*time.Second < nextDelay {
				nextDelay = 10 * time.Second
			}
			continue
		}

		delay := time.Until(dueAt)
		if delay < nextDelay {
			nextDelay = delay
		}
	}

	if !hasScheduledTask {
		return 5 * time.Minute
	}
	return nextDelay
}

func (h *Handlers) GetTasksV2(w http.ResponseWriter, r *http.Request) {
	tasks, err := db.ListTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load tasks")
		return
	}
	stats, _ := db.TaskStats()
	runs, _ := db.ListTaskRuns(200)
	taskIDs := make([]int64, 0, len(tasks))
	for _, task := range tasks {
		taskIDs = append(taskIDs, task.ID)
	}
	latestRuns, _ := db.LatestRunsByTaskIDs(taskIDs)

	tasksOut := make([]map[string]any, 0, len(tasks))
	for _, t := range tasks {
		last := latestRuns[t.ID]
		entry := map[string]any{
			"id":            t.ID,
			"name":          t.Name,
			"description":   t.Description,
			"command":       t.Command,
			"schedule_kind": t.ScheduleKind,
			"schedule_expr": t.ScheduleExpr,
			"enabled":       t.Enabled,
			"created_by":    t.CreatedBy,
			"created_at":    t.CreatedAt,
			"updated_at":    t.UpdatedAt,
		}
		if last != nil {
			entry["last_run"] = last
		}
		tasksOut = append(tasksOut, entry)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tasks":       tasksOut,
		"runs":        runs,
		"stats":       stats,
		"server_time": time.Now().Unix(),
	})
}

func (h *Handlers) CreateTaskV2(w http.ResponseWriter, r *http.Request) {
	var t db.Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := db.ValidateTask(&t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)
	t.CreatedBy = username
	id, err := db.CreateTask(&t)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create task")
		return
	}
	notifyTaskScheduler()
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "status": "created"})
}

func (h *Handlers) UpdateTaskV2(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var t db.Task
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	t.ID = id
	if err := db.ValidateTask(&t); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := db.UpdateTask(&t); err != nil {
		writeError(w, http.StatusInternalServerError, "could not update task")
		return
	}
	notifyTaskScheduler()
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated"})
}

func (h *Handlers) DeleteTaskV2(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := db.DeleteTask(id); err != nil {
		writeError(w, http.StatusInternalServerError, "could not delete task")
		return
	}
	notifyTaskScheduler()
	writeJSON(w, http.StatusOK, map[string]any{"status": "deleted"})
}

func (h *Handlers) RunTaskNow(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	t, err := db.GetTask(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)
	if !h.runTaskAsync(*t, username) {
		writeError(w, http.StatusConflict, "task already running")
		return
	}
	notifyTaskScheduler()
	writeJSON(w, http.StatusOK, map[string]any{"status": "started"})
}

func (h *Handlers) runTaskAsync(t db.Task, triggeredBy string) bool {
	taskScheduler.mu.Lock()
	if taskScheduler.running[t.ID] {
		taskScheduler.mu.Unlock()
		return false
	}
	taskScheduler.running[t.ID] = true
	taskScheduler.mu.Unlock()

	go func() {
		defer func() {
			taskScheduler.mu.Lock()
			delete(taskScheduler.running, t.ID)
			taskScheduler.mu.Unlock()
			notifyTaskScheduler()
		}()

		runID, err := db.CreateTaskRun(t.ID, triggeredBy)
		if err != nil {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		if risk, pat := classifyCommand(t.Command); risk == riskBlocked {
			_ = db.FinishTaskRun(runID, "failed", "blocked command pattern: "+pat, 1)
			return
		}

		out, err := runPrivilegedShell(ctx, t.Command)
		exitCode := 0
		status := "success"
		if err != nil {
			status = "failed"
			exitCode = 1
		}
		_ = db.FinishTaskRun(runID, status, trimOutput(out, err), exitCode)
		if status == "failed" {
			_ = db.InsertAlert("task", "warning", "tasks", fmt.Sprintf("Task '%s' failed", t.Name), "", triggeredBy)
		}
	}()
	return true
}

// runPrivilegedEnv runs a command with DEBIAN_FRONTEND=noninteractive to avoid terminal issues
func runPrivilegedEnv(ctx context.Context, cmd string, args ...string) (string, error) {
	cmdPath := lookCmd(cmd)
	if isRootUser() {
		c := exec.CommandContext(ctx, cmdPath, args...)
		// Preserve existing environment and add DEBIAN_FRONTEND
		c.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
		out, err := c.CombinedOutput()
		return string(out), err
	}
	// Use sudo with env(1) to pass DEBIAN_FRONTEND through the sudo boundary
	sudoArgs := append([]string{"-n", "env", "DEBIAN_FRONTEND=noninteractive", cmdPath}, args...)
	c := exec.CommandContext(ctx, "sudo", sudoArgs...)
	c.Env = append(os.Environ(), "DEBIAN_FRONTEND=noninteractive")
	out, err := c.CombinedOutput()
	return string(out), err
}

func runPrivileged(ctx context.Context, cmd string, args ...string) (string, error) {
	cmdPath := lookCmd(cmd)
	if isRootUser() {
		out, err := exec.CommandContext(ctx, cmdPath, args...).CombinedOutput()
		return string(out), err
	}
	sudoArgs := append([]string{"-n", cmdPath}, args...)
	out, err := exec.CommandContext(ctx, "sudo", sudoArgs...).CombinedOutput()
	return string(out), err
}

func runPrivilegedShell(ctx context.Context, command string) (string, error) {
	if isRootUser() {
		out, err := exec.CommandContext(ctx, "bash", "-lc", command).CombinedOutput()
		return string(out), err
	}
	out, err := exec.CommandContext(ctx, "sudo", "-n", "bash", "-lc", command).CombinedOutput()
	return string(out), err
}

var _isRoot = func() bool {
	u, err := user.Current()
	if err != nil {
		return false
	}
	return u.Uid == "0"
}()

func isRootUser() bool { return _isRoot }

var noPubKeyRe = regexp.MustCompile(`NO_PUBKEY ([0-9A-Fa-f]+)`)

// pkgFamily detects the system's package manager family once at startup.
var pkgFamily = func() string {
	for pm, family := range map[string]string{
		"apt-get": "apt",
		"dnf":     "dnf",
		"yum":     "yum",
		"pacman":  "pacman",
		"zypper":  "zypper",
	} {
		if _, err := exec.LookPath(pm); err == nil {
			return family
		}
	}
	return "unknown"
}()

// fixMissingRepoKeys probes the package manager for signing-key errors and
// imports any missing keys. Returns the list of key IDs (apt) or actions taken.
// Works across apt, dnf/yum, pacman, and zypper.
func fixMissingRepoKeys(ctx context.Context) []string {
	switch pkgFamily {
	case "apt":
		return fixMissingAptKeys(ctx)
	case "dnf", "yum":
		fixRPMKeys(ctx, pkgFamily)
		return nil
	case "pacman":
		runPrivileged(ctx, "pacman-key", "--refresh-keys") //nolint:errcheck
		return nil
	case "zypper":
		runPrivileged(ctx, "zypper", "--gpg-auto-import-keys", "refresh") //nolint:errcheck
		return nil
	}
	return nil
}

// fixMissingAptKeys handles apt-specific NO_PUBKEY errors.
func fixMissingAptKeys(ctx context.Context) []string {
	out, _ := runPrivileged(ctx, "apt-get", "update")
	seen := map[string]bool{}
	var imported []string

	gnupgTmp, err := os.MkdirTemp("", "scgpg-")
	if err != nil {
		return nil
	}
	defer os.RemoveAll(gnupgTmp)
	// Restrict permissions so GPG doesn't complain.
	os.Chmod(gnupgTmp, 0700) //nolint:errcheck

	for _, m := range noPubKeyRe.FindAllStringSubmatch(out, -1) {
		keyID := m[1]
		if seen[keyID] {
			continue
		}
		seen[keyID] = true

		fetched := false
		for _, ks := range []string{
			"hkps://keyserver.ubuntu.com",
			"hkps://keys.openpgp.org",
			"hkp://pool.sks-keyservers.net:80",
		} {
			_, err := runPrivilegedShell(ctx, fmt.Sprintf(
				"gpg --homedir %q --keyserver %s --recv-keys %s 2>/dev/null",
				gnupgTmp, ks, keyID,
			))
			if err == nil {
				fetched = true
				break
			}
		}
		if !fetched {
			continue
		}

		// Find the keyring the affected sources entry already references,
		// or create a per-key file as fallback.
		keyFile := fmt.Sprintf("/usr/share/keyrings/auto-%s.gpg", strings.ToLower(keyID))
		if kf, _ := runPrivilegedShell(ctx,
			`grep -rh "signed-by=" /etc/apt/sources.list.d/ 2>/dev/null | grep -oP "(?<=signed-by=)[^] ]+" | head -1`); kf != "" {
			keyFile = strings.TrimSpace(kf)
		}

		// Export the key in binary format and append to the keyring file.
		runPrivilegedShell(ctx, fmt.Sprintf( //nolint:errcheck
			`gpg --homedir %q --export %s >> %s && chmod 644 %s`,
			gnupgTmp, keyID, keyFile, keyFile,
		))

		// Patch sources entries that reference packagecloud but lack signed-by.
		runPrivilegedShell(ctx, fmt.Sprintf( //nolint:errcheck
			`for f in /etc/apt/sources.list.d/*.list; do `+
				`grep -q "packagecloud.io" "$f" 2>/dev/null && `+
				`grep -qv "signed-by" "$f" && `+
				`sed -i -E "s|^(deb(-src)?[[:space:]]+)(https?://packagecloud\.io)|\1[signed-by=%s] \3|" "$f" || true; `+
				`done`,
			keyFile,
		))
		imported = append(imported, keyID)
	}
	return imported
}

// fixRPMKeys handles dnf/yum GPG key issues by importing keys declared in repo files.
func fixRPMKeys(ctx context.Context, pm string) {
	// Collect gpgkey= URLs from all .repo files and import them.
	out, _ := runPrivilegedShell(ctx,
		`grep -rh "^gpgkey=" /etc/yum.repos.d/ 2>/dev/null | sed 's/^gpgkey=//' | sort -u`)
	for _, url := range strings.Fields(out) {
		runPrivileged(ctx, "rpm", "--import", url) //nolint:errcheck
	}
}

// sudoAvailable returns true if the process can run privileged commands.
// For non-root users it probes sudo -n true. The result is NOT cached
// because sudoers rules can change at runtime.
func sudoAvailable() bool {
	if isRootUser() {
		return true
	}
	return exec.Command("sudo", "-n", "true").Run() == nil
}

// sudoNotConfiguredMsg is the user-visible help text returned when sudo is not set up.
const sudoNotConfiguredMsg = "The sentinelcore daemon does not have permission to run privileged commands. " +
	"Install the provided sudoers configuration: sudo cp sentinelcore/deploy/sentinelcore.sudoers /etc/sudoers.d/sentinelcore && sudo chmod 440 /etc/sudoers.d/sentinelcore"

func trimOutput(out string, err error) string {
	out = strings.TrimSpace(out)
	if err == nil {
		return out
	}
	if out == "" {
		return err.Error()
	}
	if len(out) > 1000 {
		return out[:1000]
	}
	return out
}

package api

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ehsanR91/sentinelcore/internal/db"
	"github.com/ehsanR91/sentinelcore/internal/monitoring"
)

// Update state for tracking background updates
var (
	updateLogs    []string
	updateDone    bool
	updateRunning bool
	updateLastAt  time.Time
	updateMu      sync.Mutex
)

func (h *Handlers) GetSecurityStatus(w http.ResponseWriter, r *http.Request) {
	// Check which security tools are installed and running
	tools := []string{"ufw", "fail2ban", "crowdsec", "psad", "clamav-daemon",
		"auditd", "apparmor", "docker", "aide", "rkhunter"}
	services := monitoring.CheckServices(tools)

	// Recent login stats
	failed24h, _ := db.TotalRecentFailed(1440) // 24 hours, all IPs
	bans, _ := db.GetBruteForceBans(h.cfg.BruteForceThreshold, 1440)

	// Build security score using weighted health checks + recent event penalties.
	score := computeSecurityScore(services, failed24h, len(bans))

	writeJSON(w, http.StatusOK, map[string]any{
		"security_score": score,
		"services":       services,
		"failed_logins":  failed24h,
		"active_bans":    len(bans),
		"ufw_active":     isUFWActive(),
	})
}

func computeSecurityScore(services []monitoring.ServiceStatus, failed24h, activeBans int) int {
	weights := map[string]int{
		"ufw":           18,
		"fail2ban":      16,
		"crowdsec":      12,
		"psad":          8,
		"clamav-daemon": 8,
		"auditd":        14,
		"apparmor":      10,
		"docker":        4,
		"aide":          5,
		"rkhunter":      5,
	}

	score := 0
	for _, svc := range services {
		weight := weights[svc.Name]
		if weight == 0 {
			continue
		}
		if serviceHealthy(svc) {
			score += weight
		}
	}

	// Failed logins and active bans indicate active pressure and reduce posture.
	if failed24h > 0 {
		penalty := failed24h/3 + 1
		if penalty > 20 {
			penalty = 20
		}
		score -= penalty
	}
	if activeBans > 0 {
		penalty := activeBans
		if penalty > 10 {
			penalty = 10
		}
		score -= penalty
	}

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func serviceHealthy(svc monitoring.ServiceStatus) bool {
	if svc.ActiveState != "active" {
		return false
	}
	if svc.IsRunning {
		return true
	}
	// Some security controls are expected as active+exited (oneshot/unit state).
	if svc.Name == "ufw" || svc.Name == "apparmor" {
		return svc.SubState == "exited"
	}
	return false
}

func (h *Handlers) GetBans(w http.ResponseWriter, r *http.Request) {
	threshold := h.cfg.BruteForceThreshold
	if threshold <= 0 {
		threshold = 5
	}

	// Brute force bans from login_attempts
	bfBans, _ := db.GetBruteForceBans(threshold, 1440)

	// Manual bans from admin
	manualBans, _ := db.GetManualBans()
	var manualOut []map[string]any
	for _, b := range manualBans {
		manualOut = append(manualOut, map[string]any{
			"ip":        b.IP,
			"reason":    b.Reason,
			"banned_by": b.BannedBy,
			"ts":        b.Ts,
			"source":    "manual",
		})
	}

	combined := make([]map[string]any, 0)
	combined = append(combined, bfBans...)
	if manualOut != nil {
		combined = append(combined, manualOut...)
	}

	writeJSON(w, http.StatusOK, combined)
}

func (h *Handlers) Unban(w http.ResponseWriter, r *http.Request) {
	ip := chi.URLParam(r, "ip")
	if ip == "" {
		writeError(w, http.StatusBadRequest, "ip required")
		return
	}
	if !isValidIPOrCIDR(ip) {
		writeError(w, http.StatusBadRequest, "invalid IP address")
		return
	}
	db.RemoveManualBan(ip)
	_, _ = runUFW("--force", "delete", "deny", "from", ip)
	writeJSON(w, http.StatusOK, map[string]string{"status": "unbanned"})
}

func (h *Handlers) BanIP(w http.ResponseWriter, r *http.Request) {
	ip := chi.URLParam(r, "ip")
	if ip == "" {
		writeError(w, http.StatusBadRequest, "ip required")
		return
	}
	if !isValidIPOrCIDR(ip) {
		writeError(w, http.StatusBadRequest, "invalid IP address")
		return
	}
	claims := claimsFromCtx(r)
	bannedBy, _ := claims["sub"].(string)

	db.AddManualBan(ip, "manual ban", bannedBy)

	_, _ = runUFW("insert", "1", "deny", "from", ip)
	_, _ = runUFW("reload")

	writeJSON(w, http.StatusOK, map[string]string{"status": "banned"})
}

func (h *Handlers) GetUpdates(w http.ResponseWriter, r *http.Request) {
	packages := listAptUpdates()
	kernelVer := getKernelVersion()
	writeJSON(w, http.StatusOK, map[string]any{
		"packages":      packages,
		"kernel":        kernelVer,
		"count":         len(packages),
		"apt_available": aptAvailable(),
		"last_updated":  updateLastAt,
	})
}

func (h *Handlers) InstallUpdates(w http.ResponseWriter, r *http.Request) {
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	updateMu.Lock()
	if updateRunning {
		updateMu.Unlock()
		writeError(w, http.StatusConflict, "update already in progress")
		return
	}
	updateLogs = []string{}
	updateDone = false
	updateRunning = true
	updateMu.Unlock()

	go func() {
		defer func() {
			updateMu.Lock()
			updateRunning = false
			updateMu.Unlock()
		}()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()

		// Fix any repos with missing GPG keys before the real update.
		if fixed := fixMissingRepoKeys(ctx); len(fixed) > 0 {
			updateMu.Lock()
			updateLogs = append(updateLogs, fmt.Sprintf("Fixed missing GPG keys: %s", strings.Join(fixed, ", ")))
			updateMu.Unlock()
		}

		// Use privileged helper so it works whether the daemon runs as root or not.
		// APT::Update::Error-Mode=any makes individual repo failures non-fatal.
		output, err := runPrivilegedEnv(ctx, "apt-get", "-y", "-o", "APT::Update::Error-Mode=any", "update")
		updateMu.Lock()
		for _, line := range strings.Split(output, "\n") {
			if line != "" {
				updateLogs = append(updateLogs, line)
			}
		}
		if err != nil {
			updateLogs = append(updateLogs, fmt.Sprintf("Warning: apt update had errors (%v), continuing upgrade...", err))
		}
		updateMu.Unlock()

		output, err = runPrivilegedEnv(ctx, "apt-get", "-y", "upgrade")
		updateMu.Lock()
		for _, line := range strings.Split(output, "\n") {
			if line != "" {
				updateLogs = append(updateLogs, line)
			}
		}
		if err != nil {
			updateLogs = append(updateLogs, fmt.Sprintf("Error during upgrade: %v", err))
		} else {
			updateLastAt = time.Now()
			updateLogs = append(updateLogs, fmt.Sprintf("System update completed at %s", updateLastAt.Format(time.RFC3339)))
		}
		updateDone = true
		updateMu.Unlock()

		db.InsertAlert("update", "info", "system", "Package updates installed", "", "system")
	}()
	writeJSON(w, http.StatusOK, map[string]string{"status": "installing", "message": "Update running in background"})
}

// GetUpdateLogs returns the current update logs and completion status
func (h *Handlers) GetUpdateLogs(w http.ResponseWriter, r *http.Request) {
	updateMu.Lock()
	defer updateMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{
		"logs":         updateLogs,
		"done":         updateDone,
		"last_updated": updateLastAt,
	})
}

func listAptUpdates() []map[string]any {
	out, err := exec.Command("apt-get", "-s", "upgrade").Output()
	if err != nil {
		out2, err2 := exec.Command("apt", "list", "--upgradable").Output()
		if err2 != nil {
			return []map[string]any{}
		}
		return parseAptList(string(out2))
	}
	return parseAptSimulate(string(out))
}

func parseAptList(raw string) []map[string]any {
	var pkgs []map[string]any
	for _, line := range strings.Split(raw, "\n") {
		if line == "" || strings.HasPrefix(line, "Listing...") {
			continue
		}
		// Format: package/distro version arch [upgradable from: old]
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := strings.Split(parts[0], "/")[0]
		pkgs = append(pkgs, map[string]any{
			"name":        name,
			"new_ver":     parts[1],
			"old_ver":     "",
			"is_security": strings.Contains(line, "security"),
		})
	}
	if pkgs == nil {
		pkgs = []map[string]any{}
	}
	return pkgs
}

func parseAptSimulate(raw string) []map[string]any {
	var pkgs []map[string]any
	for _, line := range strings.Split(raw, "\n") {
		if !strings.HasPrefix(line, "Inst ") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 3 {
			continue
		}
		name := parts[1]
		newVer := ""
		oldVer := ""
		if len(parts) > 3 {
			newVer = strings.Trim(parts[3], "[](),")
		}
		if len(parts) > 4 {
			oldVer = strings.Trim(parts[4], "[](),")
		}
		pkgs = append(pkgs, map[string]any{
			"name":        name,
			"new_ver":     newVer,
			"old_ver":     oldVer,
			"is_security": strings.Contains(line, "security") || strings.Contains(line, "Security"),
		})
	}
	if pkgs == nil {
		pkgs = []map[string]any{}
	}
	return pkgs
}

func aptAvailable() bool {
	_, err := exec.LookPath("apt-get")
	return err == nil
}

func getKernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

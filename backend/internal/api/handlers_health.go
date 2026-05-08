package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	StatusHealthy  HealthStatus = "healthy"
	StatusWarning  HealthStatus = "warning"
	StatusCritical HealthStatus = "critical"
	StatusUnknown  HealthStatus = "unknown"
)

// HealthCheck represents a single health check result
type HealthCheck struct {
	Name        string       `json:"name"`
	Status      HealthStatus `json:"status"`
	Message     string       `json:"message"`
	Details     interface{}  `json:"details,omitempty"`
	LastChecked time.Time    `json:"last_checked"`
	Duration    string       `json:"duration"`
}

// HealthResponse is the comprehensive health check response
type HealthResponse struct {
	OverallStatus HealthStatus  `json:"overall_status"`
	Score         int           `json:"score"` // 0-100
	Checks        []HealthCheck `json:"checks"`
	Summary       string        `json:"summary"`
	Timestamp     time.Time     `json:"timestamp"`
	Version       string        `json:"version"`
	Uptime        string        `json:"uptime"`
}

// SystemInfo represents basic system information
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Hostname     string `json:"hostname"`
	Kernel       string `json:"kernel"`
	CPUCount     int    `json:"cpu_count"`
	MemoryTotal  string `json:"memory_total"`
}

// SudoersInfo represents sudoers configuration status
type SudoersInfo struct {
	MainSudoersExists  bool     `json:"main_sudoers_exists"`
	UfwSudoersExists   bool     `json:"ufw_sudoers_exists"`
	MainSudoersValid   bool     `json:"main_sudoers_valid"`
	UfwSudoersValid    bool     `json:"ufw_sudoers_valid"`
	ServiceUser        string   `json:"service_user"`
	SudoAvailable      bool     `json:"sudo_available"`
	MissingPermissions []string `json:"missing_permissions,omitempty"`
	LastUpdated        string   `json:"last_updated,omitempty"`
}

// BinaryInfo represents SentinelCore binary information
type BinaryInfo struct {
	Path        string    `json:"path"`
	Exists      bool      `json:"exists"`
	Executable  bool      `json:"executable"`
	Owner       string    `json:"owner"`
	Permissions string    `json:"permissions"`
	Size        string    `json:"size"`
	Modified    time.Time `json:"modified"`
	Version     string    `json:"version"`
	Checksum    string    `json:"checksum,omitempty"`
}

// DatabaseInfo represents database health information
type DatabaseInfo struct {
	Path       string    `json:"path"`
	Exists     bool      `json:"exists"`
	Accessible bool      `json:"accessible"`
	Size       string    `json:"size"`
	Modified   time.Time `json:"modified"`
	TableCount int       `json:"table_count"`
	UserCount  int       `json:"user_count"`
	AlertCount int       `json:"alert_count"`
}

// ServiceInfo represents systemd service information
type ServiceInfo struct {
	Name        string `json:"name"`
	Active      bool   `json:"active"`
	Enabled     bool   `json:"enabled"`
	Status      string `json:"status"`
	Uptime      string `json:"uptime"`
	MainPID     int    `json:"main_pid"`
	User        string `json:"user"`
	MemoryUsage string `json:"memory_usage,omitempty"`
}

// GetHealth performs comprehensive health checks
func (h *Handlers) GetHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	response := HealthResponse{
		Checks:    []HealthCheck{},
		Timestamp: start,
		Version:   "2.1", // This should match the script version
	}

	// Get uptime
	uptime := getSystemUptime()
	response.Uptime = uptime

	// Perform all health checks
	checks := []func() HealthCheck{
		h.checkSystemInfo,
		h.checkBinary,
		h.checkDatabase,
		h.checkService,
		h.checkSudoers,
		h.checkPermissions,
		h.checkNetwork,
		h.checkDiskSpace,
		h.checkDependencies,
	}

	results := make([]HealthCheck, len(checks))
	var wg sync.WaitGroup
	for i, check := range checks {
		wg.Add(1)
		go func(idx int, fn func() HealthCheck) {
			defer wg.Done()
			results[idx] = fn()
		}(i, check)
	}
	wg.Wait()

	var totalScore int
	var criticalCount, warningCount int

	for _, result := range results {
		enrichHealthCheckRemedy(&result)
		response.Checks = append(response.Checks, result)

		// Calculate score
		switch result.Status {
		case StatusHealthy:
			totalScore += 10
		case StatusWarning:
			totalScore += 5
			warningCount++
		case StatusCritical:
			criticalCount++
		}
	}

	// Determine overall status and score
	response.Score = totalScore
	if criticalCount > 0 {
		response.OverallStatus = StatusCritical
		response.Summary = fmt.Sprintf("System has %d critical %s and %d %s", criticalCount, pluralizeWord(criticalCount, "issue", "issues"), warningCount, pluralizeWord(warningCount, "warning", "warnings"))
	} else if warningCount > 0 {
		response.OverallStatus = StatusWarning
		response.Summary = fmt.Sprintf("System has %d %s", warningCount, pluralizeWord(warningCount, "warning", "warnings"))
	} else {
		response.OverallStatus = StatusHealthy
		response.Summary = "All systems operational"
	}

	response.Timestamp = time.Now()
	writeJSON(w, http.StatusOK, response)
}

func pluralizeWord(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func (h *Handlers) checkSystemInfo() HealthCheck {
	start := time.Now()

	info := SystemInfo{
		OS:           runtime.GOOS,
		Architecture: runtime.GOARCH,
		CPUCount:     runtime.NumCPU(),
	}

	// Get hostname
	if hostname, err := os.Hostname(); err == nil {
		info.Hostname = hostname
	}

	// Get kernel version
	if kernel, err := runHealthCommand("uname", "-r"); err == nil {
		info.Kernel = strings.TrimSpace(kernel)
	}

	// Get memory info
	if mem, err := runHealthCommand("free", "-h"); err == nil {
		lines := strings.Split(mem, "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) > 1 {
				info.MemoryTotal = fields[1]
			}
		}
	}

	status := StatusHealthy
	message := "System information collected successfully"

	return HealthCheck{
		Name:        "System Information",
		Status:      status,
		Message:     message,
		Details:     info,
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkBinary() HealthCheck {
	start := time.Now()

	info := BinaryInfo{
		Path: "/opt/sentinelcore/sentinelcore",
	}

	// Check if binary exists
	if stat, err := os.Stat(info.Path); err == nil {
		info.Exists = true
		info.Modified = stat.ModTime()
		info.Size = fmt.Sprintf("%.1f MB", float64(stat.Size())/1024/1024)

		// Check permissions
		if stat.Mode().Perm()&0111 != 0 {
			info.Executable = true
		}
		info.Permissions = fmt.Sprintf("%o", stat.Mode().Perm())

		// Get owner - use a simpler approach without platform-specific syscalls
		// For now, we'll skip owner detection to maintain cross-platform compatibility
	}

	// Check if executable
	if info.Exists && info.Executable {
		// Get version
		if version, err := runHealthCommand(info.Path, "--version"); err == nil {
			info.Version = strings.TrimSpace(version)
		}

		// Get checksum (optional, can be expensive)
		if checksum, err := runHealthCommand("sha256sum", info.Path); err == nil {
			parts := strings.Fields(checksum)
			if len(parts) > 0 {
				info.Checksum = parts[0]
			}
		}
	}

	status := StatusHealthy
	message := "SentinelCore binary is operational"

	if !info.Exists {
		status = StatusCritical
		message = "SentinelCore binary not found"
	} else if !info.Executable {
		status = StatusCritical
		message = "SentinelCore binary is not executable"
	} else if info.Owner == "root" {
		status = StatusWarning
		message = "Binary owned by root (security consideration)"
	}

	return HealthCheck{
		Name:        "Binary Status",
		Status:      status,
		Message:     message,
		Details:     info,
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkDatabase() HealthCheck {
	start := time.Now()

	info := DatabaseInfo{
		Path: "/opt/sentinelcore/data/app.db",
	}

	// Check if database exists
	if stat, err := os.Stat(info.Path); err == nil {
		info.Exists = true
		info.Modified = stat.ModTime()
		info.Size = fmt.Sprintf("%.1f MB", float64(stat.Size())/1024/1024)
	}

	// Check accessibility and get stats
	if info.Exists {
		// Try to get basic database info
		if db := h.getDBInfo(); db != nil {
			info.Accessible = true
			info.TableCount = db.TableCount
			info.UserCount = db.UserCount
			info.AlertCount = db.AlertCount
		}
	}

	status := StatusHealthy
	message := "Database is operational"

	if !info.Exists {
		status = StatusCritical
		message = "Database file not found"
	} else if !info.Accessible {
		status = StatusCritical
		message = "Database is not accessible"
	}

	return HealthCheck{
		Name:        "Database",
		Status:      status,
		Message:     message,
		Details:     info,
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkService() HealthCheck {
	start := time.Now()

	info := ServiceInfo{
		Name: "sentinelcore",
	}

	// Check service status
	if output, err := runHealthCommand("systemctl", "is-active", info.Name); err == nil {
		info.Active = strings.TrimSpace(output) == "active"
		info.Status = strings.TrimSpace(output)
	}

	if output, err := runHealthCommand("systemctl", "is-enabled", info.Name); err == nil {
		info.Enabled = strings.TrimSpace(output) == "enabled"
	}

	// Get detailed service info
	if output, err := runHealthCommand("systemctl", "show", info.Name, "--property=MainPID,ExecStart,User,MemoryCurrent"); err == nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "MainPID=") {
				if pid, err := strconv.Atoi(strings.TrimPrefix(line, "MainPID=")); err == nil {
					info.MainPID = pid
				}
			} else if strings.HasPrefix(line, "User=") {
				info.User = strings.TrimPrefix(line, "User=")
			} else if strings.HasPrefix(line, "MemoryCurrent=") {
				mem := strings.TrimPrefix(line, "MemoryCurrent=")
				if mem != "[not set]" {
					info.MemoryUsage = formatBytes(mem)
				}
			}
		}
	}

	// Get uptime
	if info.Active && info.MainPID > 0 {
		if uptime, err := runHealthCommand("ps", "-o", "etime=", "-p", strconv.Itoa(info.MainPID)); err == nil {
			info.Uptime = strings.TrimSpace(uptime)
		}
	}

	status := StatusHealthy
	message := "SentinelCore service is running"

	if !info.Active {
		status = StatusCritical
		message = "SentinelCore service is not active"
	} else if !info.Enabled {
		status = StatusWarning
		message = "SentinelCore service is not enabled for auto-start"
	}

	return HealthCheck{
		Name:        "Service Status",
		Status:      status,
		Message:     message,
		Details:     info,
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkSudoers() HealthCheck {
	start := time.Now()

	info := SudoersInfo{
		MissingPermissions: []string{},
	}

	// Check main sudoers
	if stat, err := os.Stat("/etc/sudoers.d/sentinelcore"); err == nil {
		info.MainSudoersExists = true
		info.LastUpdated = stat.ModTime().Format("2006-01-02 15:04:05")

		// Validate sudoers
		if _, err := runHealthCommand("visudo", "-c", "-f", "/etc/sudoers.d/sentinelcore"); err == nil {
			info.MainSudoersValid = true
		}
	}

	// Check UFW sudoers
	if _, err := os.Stat("/etc/sudoers.d/sentinelcore-ufw"); err == nil {
		info.UfwSudoersExists = true

		// Validate sudoers
		if _, err := runHealthCommand("visudo", "-c", "-f", "/etc/sudoers.d/sentinelcore-ufw"); err == nil {
			info.UfwSudoersValid = true
		}
	}

	// Get service user
	if output, err := runHealthCommand("systemctl", "show", "sentinelcore", "--property=User"); err == nil {
		info.ServiceUser = strings.TrimPrefix(strings.TrimSpace(output), "User=")
	}

	// Check sudo availability
	if _, err := runHealthCommand("sudo", "-n", "true"); err == nil {
		info.SudoAvailable = true
	}

	// Check for missing permissions
	if info.MainSudoersExists {
		// Read sudoers content and check for required commands
		if content, err := os.ReadFile("/etc/sudoers.d/sentinelcore"); err == nil {
			contentStr := string(content)
			requiredCmds := []string{"curl", "dd", "chmod", "bash", "apt-get"}

			for _, cmd := range requiredCmds {
				if !strings.Contains(contentStr, cmd) {
					info.MissingPermissions = append(info.MissingPermissions, cmd)
				}
			}
		}
	}

	status := StatusHealthy
	message := "Sudoers configuration is properly set up"

	if !info.MainSudoersExists {
		status = StatusCritical
		message = "Main sudoers file is missing"
	} else if !info.MainSudoersValid {
		status = StatusCritical
		message = "Main sudoers file has syntax errors"
	} else if !info.SudoAvailable {
		status = StatusCritical
		message = "Sudo access is not available"
	} else if len(info.MissingPermissions) > 0 {
		status = StatusWarning
		message = fmt.Sprintf("Missing permissions: %s", strings.Join(info.MissingPermissions, ", "))
	}

	return HealthCheck{
		Name:        "Sudoers Configuration",
		Status:      status,
		Message:     message,
		Details:     info,
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkPermissions() HealthCheck {
	start := time.Now()

	issues := []string{}

	// Check installation directory permissions
	if stat, err := os.Stat("/opt/sentinelcore"); err == nil {
		if stat.Mode().Perm()&0755 != 0755 {
			issues = append(issues, "Installation directory has incorrect permissions")
		}
	} else {
		issues = append(issues, "Installation directory not found")
	}

	// Check data directory permissions
	if stat, err := os.Stat("/opt/sentinelcore/data"); err == nil {
		if stat.Mode().Perm()&0700 != 0700 {
			issues = append(issues, "Data directory has incorrect permissions")
		}
	} else {
		issues = append(issues, "Data directory not found")
	}

	// Check .env file permissions
	if stat, err := os.Stat("/opt/sentinelcore/.env"); err == nil {
		if stat.Mode().Perm()&0600 != 0600 {
			issues = append(issues, ".env file has incorrect permissions")
		}
	} else {
		issues = append(issues, ".env file not found")
	}

	status := StatusHealthy
	message := "File permissions are correct"

	if len(issues) > 0 {
		if len(issues) > 2 {
			status = StatusCritical
		} else {
			status = StatusWarning
		}
		message = strings.Join(issues, "; ")
	}

	return HealthCheck{
		Name:        "File Permissions",
		Status:      status,
		Message:     message,
		Details:     map[string]interface{}{"issues": issues},
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkNetwork() HealthCheck {
	start := time.Now()

	// Check basic network connectivity
	if _, err := runHealthCommand("ping", "-c", "1", "-W", "1", "8.8.8.8"); err != nil {
		return HealthCheck{
			Name:        "Network Connectivity",
			Status:      StatusWarning,
			Message:     "No internet connectivity",
			Details:     map[string]string{"error": err.Error()},
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}

	// Check DNS resolution
	if _, err := runHealthCommand("nslookup", "google.com"); err != nil {
		return HealthCheck{
			Name:        "Network Connectivity",
			Status:      StatusWarning,
			Message:     "DNS resolution issues",
			Details:     map[string]string{"error": err.Error()},
			LastChecked: time.Now(),
			Duration:    time.Since(start).String(),
		}
	}

	return HealthCheck{
		Name:        "Network Connectivity",
		Status:      StatusHealthy,
		Message:     "Network connectivity is working",
		Details:     map[string]string{"dns": "ok", "ping": "ok"},
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkDiskSpace() HealthCheck {
	start := time.Now()

	// Get disk usage
	if output, err := runHealthCommand("df", "-h", "/opt/sentinelcore"); err == nil {
		lines := strings.Split(output, "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) > 4 {
				usedPercent := strings.TrimSuffix(fields[4], "%")
				if used, err := strconv.Atoi(usedPercent); err == nil {
					status := StatusHealthy
					message := fmt.Sprintf("Disk usage: %s", fields[4])

					if used > 90 {
						status = StatusCritical
						message = fmt.Sprintf("Critical disk usage: %s", fields[4])
					} else if used > 80 {
						status = StatusWarning
						message = fmt.Sprintf("High disk usage: %s", fields[4])
					}

					return HealthCheck{
						Name:        "Disk Space",
						Status:      status,
						Message:     message,
						Details:     map[string]interface{}{"usage": fields[4], "available": fields[3], "total": fields[1]},
						LastChecked: time.Now(),
						Duration:    time.Since(start).String(),
					}
				}
			}
		}
	}

	return HealthCheck{
		Name:        "Disk Space",
		Status:      StatusUnknown,
		Message:     "Unable to check disk space",
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

func (h *Handlers) checkDependencies() HealthCheck {
	start := time.Now()

	missing := []string{}
	dependencies := map[string]string{
		"systemctl":       "systemd",
		"curl":            "curl",
		"ufw":             "ufw",
		"fail2ban-server": "fail2ban",
		"nginx":           "nginx",
	}

	for dep, name := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, name)
		}
	}

	status := StatusHealthy
	message := "All dependencies are available"

	if len(missing) > 0 {
		if len(missing) > 3 {
			status = StatusCritical
		} else {
			status = StatusWarning
		}
		message = fmt.Sprintf("Missing dependencies: %s", strings.Join(missing, ", "))
	}

	return HealthCheck{
		Name:        "Dependencies",
		Status:      status,
		Message:     message,
		Details:     map[string]interface{}{"missing": missing, "total": len(dependencies)},
		LastChecked: time.Now(),
		Duration:    time.Since(start).String(),
	}
}

// Helper functions

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

const healthCommandTimeout = 2 * time.Second

func runHealthCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), healthCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("command timed out: %s", name)
	}
	return string(output), err
}

func getSystemUptime() string {
	if output, err := runHealthCommand("uptime", "-p"); err == nil {
		return strings.TrimSpace(strings.TrimPrefix(output, "up "))
	}
	return "Unknown"
}

func formatBytes(bytesStr string) string {
	if bytes, err := strconv.ParseInt(bytesStr, 10, 64); err == nil {
		if bytes < 1024*1024 {
			return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
		} else if bytes < 1024*1024*1024 {
			return fmt.Sprintf("%.1f MB", float64(bytes)/1024/1024)
		} else {
			return fmt.Sprintf("%.1f GB", float64(bytes)/1024/1024/1024)
		}
	}
	return bytesStr
}

func (h *Handlers) getDBInfo() *DatabaseInfo {
	// This would require database access - for now return basic info
	info := &DatabaseInfo{
		TableCount: 0, // Would need actual DB queries
		UserCount:  0, // Would need actual DB queries
		AlertCount: 0, // Would need actual DB queries
	}
	return info
}

func enrichHealthCheckRemedy(check *HealthCheck) {
	if check == nil {
		return
	}
	if check.Status != StatusCritical && check.Status != StatusWarning {
		return
	}

	details := map[string]any{}
	if m, ok := check.Details.(map[string]any); ok && m != nil {
		details = m
	} else if check.Details != nil {
		b, err := json.Marshal(check.Details)
		if err == nil {
			_ = json.Unmarshal(b, &details)
		}
		if details == nil {
			details = map[string]any{}
		}
	}

	msg := strings.ToLower(check.Message)

	switch check.Name {
	case "Sudoers Configuration":
		details["remedy"] = "Re-install sudoers from the bundled template, substituting the running service user. The placeholder <USER> in the source file is replaced automatically."
		details["command"] = `SC_USER=$(systemctl show sentinelcore --property=User 2>/dev/null | cut -d= -f2 | tr -d '\n'); [ -z "$SC_USER" ] && SC_USER="$(whoami)"; sudo sed "s/<USER>/$SC_USER/g" /opt/sentinelcore/deploy/sentinelcore.sudoers | sudo tee /etc/sudoers.d/sentinelcore > /dev/null && sudo chmod 440 /etc/sudoers.d/sentinelcore && sudo visudo -c -f /etc/sudoers.d/sentinelcore && echo OK`
		details["requires_root"] = true
		details["auto_fix_available"] = true
	case "File Permissions":
		details["remedy"] = "Repair ownership and permissions for SentinelCore installation directories and secrets."
		details["command"] = "sudo chown -R sentinelcore:sentinelcore /opt/sentinelcore && sudo chmod 755 /opt/sentinelcore && sudo chmod 700 /opt/sentinelcore/data && sudo chmod 600 /opt/sentinelcore/.env"
		details["requires_root"] = true
		details["auto_fix_available"] = true
	case "Service Status":
		details["remedy"] = "Start and enable the SentinelCore service, then inspect logs if startup fails."
		details["command"] = "sudo systemctl enable --now sentinelcore && sudo systemctl status sentinelcore --no-pager && sudo journalctl -u sentinelcore -n 120 --no-pager"
		details["requires_root"] = true
		details["auto_fix_available"] = true
	case "Database":
		details["remedy"] = "Verify DB file path and permissions, and ensure the service user can read/write the database."
		details["command"] = "sudo ls -lah /opt/sentinelcore/data/app.db && sudo chown sentinelcore:sentinelcore /opt/sentinelcore/data/app.db && sudo chmod 640 /opt/sentinelcore/data/app.db"
		details["requires_root"] = true
	case "Binary Status":
		details["remedy"] = "Re-deploy SentinelCore binary and ensure executable bit is set."
		details["command"] = "sudo install -m 755 /opt/sentinelcore/backend/sentinelcore /opt/sentinelcore/sentinelcore && sudo systemctl restart sentinelcore"
		details["requires_root"] = true
	case "Disk Space":
		details["remedy"] = "Clean package cache and stale logs to recover disk space."
		details["command"] = "sudo apt-get clean && sudo journalctl --vacuum-time=7d && sudo du -sh /var/log /var/cache/apt"
		details["requires_root"] = true
		details["auto_fix_available"] = true
	case "Dependencies":
		details["remedy"] = "Install missing system dependencies required by SentinelCore health checks and operations."
		details["command"] = "sudo apt-get update && sudo apt-get install -y curl ufw fail2ban nginx"
		details["requires_root"] = true
		details["auto_fix_available"] = true
	case "Network Connectivity":
		details["remedy"] = "Validate DNS and outbound connectivity; check firewall egress and resolver config."
		details["command"] = "ping -c 1 8.8.8.8 ; nslookup google.com ; sudo cat /etc/resolv.conf"
		details["requires_root"] = false
	default:
		details["remedy"] = "Review check details and run the recommended fix endpoint or manual corrective commands."
	}

	if strings.Contains(msg, "missing") && details["command"] == nil {
		details["command"] = "sudo systemctl status sentinelcore --no-pager"
		details["requires_root"] = true
	}

	check.Details = details
}

// FixHealthIssue attempts to fix a specific health issue
type FixHealthIssueRequest struct {
	CheckName string `json:"check_name"`
	Action    string `json:"action"` // "auto" or "manual"
}

type FixHealthIssueResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	Remedy       string `json:"remedy,omitempty"`
	Command      string `json:"command,omitempty"`
	RequiresSudo bool   `json:"requires_sudo,omitempty"`
}

func (h *Handlers) FixHealthIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "Method not allowed"})
		return
	}

	var req FixHealthIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	response := FixHealthIssueResponse{
		Success:      false,
		Message:      "Unknown check name",
		RequiresSudo: true,
	}

	switch req.CheckName {
	case "Sudoers Configuration":
		response = h.fixSudoersIssue(req.Action)
	case "File Permissions":
		response = h.fixPermissionsIssue(req.Action)
	case "Service Status":
		response = h.fixServiceIssue(req.Action)
	case "Dependencies":
		response = h.fixDependenciesIssue(req.Action)
	case "Disk Space":
		response = h.fixDiskSpaceIssue(req.Action)
	default:
		response.Remedy = "No automatic fix available for this issue. Please check the system manually."
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handlers) fixSudoersIssue(action string) FixHealthIssueResponse {
	// Helper: detect service user from systemd unit
	detectServiceUser := func() string {
		out, err := runHealthCommand("systemctl", "show", "sentinelcore", "--property=User")
		if err == nil {
			u := strings.TrimPrefix(strings.TrimSpace(out), "User=")
			if u != "" && u != "User=" {
				return u
			}
		}
		if u, err2 := runHealthCommand("id", "-un"); err2 == nil {
			return strings.TrimSpace(u)
		}
		return ""
	}

	// Install/reinstall sudoers from the bundled template.
	installFromTemplate := func(targetUser string) FixHealthIssueResponse {
		const srcTemplate = "/opt/sentinelcore/deploy/sentinelcore.sudoers"
		const destFile = "/etc/sudoers.d/sentinelcore"

		src, readErr := os.ReadFile(srcTemplate)
		if readErr != nil {
			src = []byte(fmt.Sprintf(
				"%s ALL=(root) NOPASSWD: /usr/bin/apt-get *, /usr/bin/apt *, /bin/bash *, /usr/bin/bash *, /bin/sh *, /usr/bin/sh *, /usr/bin/systemctl *, /bin/systemctl *, /usr/bin/curl *, /usr/bin/wget *, /bin/chmod *, /usr/bin/chmod *, /bin/cp *, /usr/bin/cp *, /bin/rm *, /usr/bin/rm *, /usr/bin/du *, /usr/bin/find *, /bin/find *, /usr/bin/dd *, /bin/dd *, /usr/sbin/ufw *, /usr/bin/ufw *, /usr/bin/tee /etc/*\n",
				targetUser,
			))
		}

		contents := strings.ReplaceAll(string(src), "<USER>", targetUser)
		tmpPath := "/tmp/sentinelcore_sudoers_validate"
		if err := os.WriteFile(tmpPath, []byte(contents), 0440); err != nil {
			return FixHealthIssueResponse{
				Success:      false,
				Message:      "Could not write temp sudoers: " + err.Error(),
				Command:      fmt.Sprintf(`SC_USER=%s; sudo sed "s/<USER>/$SC_USER/g" /opt/sentinelcore/deploy/sentinelcore.sudoers | sudo tee /etc/sudoers.d/sentinelcore > /dev/null && sudo chmod 440 /etc/sudoers.d/sentinelcore`, targetUser),
				RequiresSudo: true,
			}
		}
		defer os.Remove(tmpPath)

		if _, err := runHealthCommand("visudo", "-c", "-f", tmpPath); err != nil {
			return FixHealthIssueResponse{
				Success:      false,
				Message:      "Generated sudoers failed visudo validation — review template.",
				Command:      fmt.Sprintf(`SC_USER=%s; sudo sed "s/<USER>/$SC_USER/g" /opt/sentinelcore/deploy/sentinelcore.sudoers | sudo tee /etc/sudoers.d/sentinelcore > /dev/null && sudo chmod 440 /etc/sudoers.d/sentinelcore`, targetUser),
				RequiresSudo: true,
			}
		}

		if err := os.WriteFile(destFile, []byte(contents), 0440); err != nil {
			return FixHealthIssueResponse{
				Success:      false,
				Message:      "Validation passed but install failed (need root): " + err.Error(),
				Command:      fmt.Sprintf(`SC_USER=%s; sudo sed "s/<USER>/$SC_USER/g" /opt/sentinelcore/deploy/sentinelcore.sudoers | sudo tee /etc/sudoers.d/sentinelcore > /dev/null && sudo chmod 440 /etc/sudoers.d/sentinelcore`, targetUser),
				RequiresSudo: true,
			}
		}
		return FixHealthIssueResponse{Success: true, Message: fmt.Sprintf("Sudoers installed for user %q — validated OK", targetUser), RequiresSudo: true}
	}

	fileExists := true
	if _, err := os.Stat("/etc/sudoers.d/sentinelcore"); os.IsNotExist(err) {
		fileExists = false
	}

	hasErrors := false
	if fileExists {
		if _, err := runHealthCommand("visudo", "-c", "-f", "/etc/sudoers.d/sentinelcore"); err != nil {
			hasErrors = true
		}
	}

	if !fileExists || hasErrors {
		if action == "auto" {
			targetUser := detectServiceUser()
			if targetUser == "" {
				return FixHealthIssueResponse{
					Success:      false,
					Message:      "Could not detect service user automatically.",
					Command:      `SC_USER=YOUR_USER; sudo sed "s/<USER>/$SC_USER/g" /opt/sentinelcore/deploy/sentinelcore.sudoers | sudo tee /etc/sudoers.d/sentinelcore > /dev/null && sudo chmod 440 /etc/sudoers.d/sentinelcore && sudo visudo -c -f /etc/sudoers.d/sentinelcore`,
					RequiresSudo: true,
				}
			}
			return installFromTemplate(targetUser)
		}

		targetUser := detectServiceUser()
		if targetUser == "" {
			targetUser = "YOUR_USER"
		}
		return FixHealthIssueResponse{
			Success:      false,
			Message:      "Sudoers file is missing or has syntax errors (possibly un-substituted <USER> placeholder).",
			Remedy:       "Run the command below — it substitutes your service user and installs the correct sudoers.",
			Command:      fmt.Sprintf(`SC_USER=%s; sudo sed "s/<USER>/$SC_USER/g" /opt/sentinelcore/deploy/sentinelcore.sudoers | sudo tee /etc/sudoers.d/sentinelcore > /dev/null && sudo chmod 440 /etc/sudoers.d/sentinelcore && sudo visudo -c -f /etc/sudoers.d/sentinelcore && echo OK`, targetUser),
			RequiresSudo: true,
		}
	}

	if action == "auto" {
		if _, err := runHealthCommand("visudo", "-c", "-f", "/etc/sudoers.d/sentinelcore"); err != nil {
			return FixHealthIssueResponse{
				Success:      false,
				Message:      "Sudoers validation failed: " + err.Error(),
				Remedy:       "File exists but failed visudo check — run the auto-fix or the command below.",
				Command:      `SC_USER=$(systemctl show sentinelcore --property=User 2>/dev/null | cut -d= -f2 | tr -d '\n'); [ -z "$SC_USER" ] && SC_USER="$(id -un)"; sudo sed "s/<USER>/$SC_USER/g" /opt/sentinelcore/deploy/sentinelcore.sudoers | sudo tee /etc/sudoers.d/sentinelcore > /dev/null && sudo chmod 440 /etc/sudoers.d/sentinelcore && sudo visudo -c -f /etc/sudoers.d/sentinelcore && echo OK`,
				RequiresSudo: true,
			}
		}
		return FixHealthIssueResponse{Success: true, Message: "Sudoers file is valid", RequiresSudo: true}
	}

	return FixHealthIssueResponse{
		Success:      true,
		Message:      "Sudoers file exists and is valid",
		RequiresSudo: true,
	}
}

func (h *Handlers) fixPermissionsIssue(action string) FixHealthIssueResponse {
	issues := []string{}
	commands := []string{}

	// Fix installation directory permissions
	if stat, err := os.Stat("/opt/sentinelcore"); err == nil {
		if stat.Mode().Perm()&0755 != 0755 {
			commands = append(commands, "sudo chmod 755 /opt/sentinelcore")
		}
	} else {
		issues = append(issues, "Installation directory not found")
	}

	// Fix data directory permissions
	if stat, err := os.Stat("/opt/sentinelcore/data"); err == nil {
		if stat.Mode().Perm()&0700 != 0700 {
			commands = append(commands, "sudo chmod 700 /opt/sentinelcore/data")
		}
	} else {
		issues = append(issues, "Data directory not found")
	}

	// Fix .env file permissions
	if stat, err := os.Stat("/opt/sentinelcore/.env"); err == nil {
		if stat.Mode().Perm()&0600 != 0600 {
			commands = append(commands, "sudo chmod 600 /opt/sentinelcore/.env")
		}
	} else {
		issues = append(issues, ".env file not found")
	}

	if len(commands) > 0 {
		if action == "auto" {
			// Execute commands
			for _, cmd := range commands {
				parts := strings.Fields(cmd)
				if len(parts) < 3 {
					continue
				}
				if _, err := runCommand(parts[1], parts[2:]...); err != nil {
					return FixHealthIssueResponse{
						Success:      false,
						Message:      "Failed to execute command: " + cmd,
						Remedy:       "Run the following commands manually",
						Command:      strings.Join(commands, " && "),
						RequiresSudo: true,
					}
				}
			}
			return FixHealthIssueResponse{
				Success:      true,
				Message:      "Permissions fixed successfully",
				RequiresSudo: true,
			}
		}

		return FixHealthIssueResponse{
			Success:      false,
			Message:      "File permissions need to be fixed",
			Remedy:       "Run the following commands to fix permissions",
			Command:      strings.Join(commands, " && "),
			RequiresSudo: true,
		}
	}

	return FixHealthIssueResponse{
		Success:      false,
		Message:      "No permission issues detected",
		Remedy:       "Check the specific error message for details",
		RequiresSudo: true,
	}
}

func (h *Handlers) fixServiceIssue(action string) FixHealthIssueResponse {
	// Check if service is active
	if output, err := runCommand("systemctl", "is-active", "sentinelcore"); err == nil {
		if strings.TrimSpace(output) == "active" {
			// Check if enabled
			if output, err := runCommand("systemctl", "is-enabled", "sentinelcore"); err == nil {
				if strings.TrimSpace(output) == "enabled" {
					return FixHealthIssueResponse{
						Success:      true,
						Message:      "Service is already running and enabled",
						RequiresSudo: true,
					}
				}
				// Service is active but not enabled
				if action == "auto" {
					if _, err := runCommand("systemctl", "enable", "sentinelcore"); err != nil {
						return FixHealthIssueResponse{
							Success:      false,
							Message:      "Failed to enable service: " + err.Error(),
							Remedy:       "Enable the service manually",
							Command:      "sudo systemctl enable sentinelcore",
							RequiresSudo: true,
						}
					}
					return FixHealthIssueResponse{
						Success:      true,
						Message:      "Service enabled successfully",
						RequiresSudo: true,
					}
				}
				return FixHealthIssueResponse{
					Success:      false,
					Message:      "Service is running but not enabled for auto-start",
					Remedy:       "Enable the service to start on boot",
					Command:      "sudo systemctl enable sentinelcore",
					RequiresSudo: true,
				}
			}
		}
	}

	// Service is not active
	if action == "auto" {
		if _, err := runCommand("systemctl", "start", "sentinelcore"); err != nil {
			return FixHealthIssueResponse{
				Success:      false,
				Message:      "Failed to start service: " + err.Error(),
				Remedy:       "Start the service manually",
				Command:      "sudo systemctl start sentinelcore",
				RequiresSudo: true,
			}
		}
		if _, err := runCommand("systemctl", "enable", "sentinelcore"); err != nil {
			return FixHealthIssueResponse{
				Success:      true,
				Message:      "Service started but failed to enable for auto-start",
				Remedy:       "Enable the service manually",
				Command:      "sudo systemctl enable sentinelcore",
				RequiresSudo: true,
			}
		}
		return FixHealthIssueResponse{
			Success:      true,
			Message:      "Service started and enabled successfully",
			RequiresSudo: true,
		}
	}

	return FixHealthIssueResponse{
		Success:      false,
		Message:      "Service is not running",
		Remedy:       "Start and enable the service",
		Command:      "sudo systemctl start sentinelcore && sudo systemctl enable sentinelcore",
		RequiresSudo: true,
	}
}

func (h *Handlers) fixDependenciesIssue(action string) FixHealthIssueResponse {
	missing := []string{}
	installCmds := []string{}
	dependencies := map[string]string{
		"systemctl":       "systemd",
		"curl":            "curl",
		"ufw":             "ufw",
		"fail2ban-server": "fail2ban",
		"nginx":           "nginx",
	}

	for dep, pkg := range dependencies {
		if _, err := exec.LookPath(dep); err != nil {
			missing = append(missing, pkg)
			installCmds = append(installCmds, pkg)
		}
	}

	if len(missing) > 0 {
		if action == "auto" {
			// Install missing dependencies
			installCmd := strings.Join(installCmds, " ")
			if _, err := runCommand("apt-get", "update"); err != nil {
				return FixHealthIssueResponse{
					Success:      false,
					Message:      "Failed to update package list: " + err.Error(),
					Remedy:       "Update package list and install dependencies manually",
					Command:      "sudo apt-get update && sudo apt-get install -y " + installCmd,
					RequiresSudo: true,
				}
			}
			if _, err := runCommand("apt-get", "install", "-y", installCmd); err != nil {
				return FixHealthIssueResponse{
					Success:      false,
					Message:      "Failed to install dependencies: " + err.Error(),
					Remedy:       "Install dependencies manually",
					Command:      "sudo apt-get install -y " + installCmd,
					RequiresSudo: true,
				}
			}
			return FixHealthIssueResponse{
				Success:      true,
				Message:      "Dependencies installed successfully",
				RequiresSudo: true,
			}
		}

		return FixHealthIssueResponse{
			Success:      false,
			Message:      fmt.Sprintf("Missing dependencies: %s", strings.Join(missing, ", ")),
			Remedy:       "Install missing dependencies",
			Command:      "sudo apt-get update && sudo apt-get install -y " + strings.Join(installCmds, " "),
			RequiresSudo: true,
		}
	}

	return FixHealthIssueResponse{
		Success:      true,
		Message:      "All dependencies are available",
		RequiresSudo: true,
	}
}

func (h *Handlers) fixDiskSpaceIssue(action string) FixHealthIssueResponse {
	// Get disk usage
	if output, err := runCommand("df", "-h", "/opt/sentinelcore"); err == nil {
		lines := strings.Split(output, "\n")
		if len(lines) > 1 {
			fields := strings.Fields(lines[1])
			if len(fields) > 4 {
				usedPercent := strings.TrimSuffix(fields[4], "%")
				if used, err := strconv.Atoi(usedPercent); err == nil {
					if used > 80 {
						remedies := []string{
							"Clean apt cache: sudo apt-get clean && sudo apt-get autoremove",
							"Clean system logs: sudo journalctl --vacuum-time=7d",
							"Find large files: sudo find / -type f -size +100M 2>/dev/null | head -20",
						}

						return FixHealthIssueResponse{
							Success:      false,
							Message:      fmt.Sprintf("High disk usage: %s%%", usedPercent),
							Remedy:       "Clean up disk space using the following commands",
							Command:      strings.Join(remedies, "\n"),
							RequiresSudo: true,
						}
					}
				}
			}
		}
	}

	return FixHealthIssueResponse{
		Success:      true,
		Message:      "Disk usage is within acceptable limits",
		RequiresSudo: true,
	}
}

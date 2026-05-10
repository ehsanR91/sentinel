package api

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ehsanR91/sentinelcore/internal/db"
)

type securityToolDef struct {
	Name        string
	Label       string
	Description string
	Command     string
	Timeout     time.Duration
}

var securityToolCatalog = []securityToolDef{
	{
		Name:        "rkhunter",
		Label:       "rkhunter",
		Description: "Rootkit Hunter deep scan",
		Command:     "rkhunter --check --sk --nocolors",
		Timeout:     45 * time.Minute,
	},
	{
		Name:        "clamav",
		Label:       "ClamAV",
		Description: "Fresh signature update + malware scan",
		Command: `if command -v freshclam >/dev/null 2>&1; then
  freshclam --quiet 2>&1 || echo "[WARN] freshclam update failed, continuing with current definitions"
else
  echo "[WARN] freshclam is not installed, skipping signature update"
fi
clamscan -r /etc --max-filesize=25M --infected --no-summary`,
		Timeout: 45 * time.Minute,
	},
	{
		Name:        "lynis",
		Label:       "Lynis",
		Description: "System hardening audit",
		Command:     "lynis audit system --quick",
		Timeout:     45 * time.Minute,
	},
	{
		Name:        "chkrootkit",
		Label:       "chkrootkit",
		Description: "Rootkit signature checks",
		Command:     "chkrootkit",
		Timeout:     30 * time.Minute,
	},
}

type toolRunState struct {
	Running    bool     `json:"running"`
	LastStatus string   `json:"last_status"`
	LastRunAt  string   `json:"last_run_at"`
	LastError  string   `json:"last_error,omitempty"`
	Logs       []string `json:"logs"`
}

var (
	toolRunMu    sync.Mutex
	toolRunStore = map[string]*toolRunState{}
)

var (
	cleanupMu        sync.Mutex
	cleanupRunning   bool
	cleanupDone      bool
	cleanupLogs      []string
	cleanupLastAt    time.Time
	cleanupLastFreed int64
)

func init() {
	for _, t := range securityToolCatalog {
		toolRunStore[t.Name] = &toolRunState{LastStatus: "never", Logs: []string{}}
	}
}

func (h *Handlers) GetCleanupStats(w http.ResponseWriter, r *http.Request) {
	est, _ := estimateCleanupBytes(r.Context())
	cleanupMu.Lock()
	resp := map[string]any{
		"estimated_junk_bytes": est,
		"estimated_junk_human": humanBytes(uint64(est)),
		"running":              cleanupRunning,
		"done":                 cleanupDone,
		"last_cleaned_at":      cleanupLastAt,
		"last_freed_bytes":     cleanupLastFreed,
		"last_freed_human":     humanBytes(uint64(cleanupLastFreed)),
	}
	cleanupMu.Unlock()
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) GetCleanupLogs(w http.ResponseWriter, r *http.Request) {
	cleanupMu.Lock()
	resp := map[string]any{
		"logs":             append([]string{}, cleanupLogs...),
		"running":          cleanupRunning,
		"done":             cleanupDone,
		"last_cleaned_at":  cleanupLastAt,
		"last_freed_bytes": cleanupLastFreed,
		"last_freed_human": humanBytes(uint64(cleanupLastFreed)),
	}
	cleanupMu.Unlock()
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) RunCleanup(w http.ResponseWriter, r *http.Request) {
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}

	cleanupMu.Lock()
	if cleanupRunning {
		cleanupMu.Unlock()
		writeError(w, http.StatusConflict, "cleanup already running")
		return
	}
	cleanupRunning = true
	cleanupDone = false
	cleanupLogs = []string{fmt.Sprintf("[%s] Starting system cleanup", time.Now().Format(time.RFC3339))}
	cleanupMu.Unlock()

	go func() {
		start := time.Now()
		before, _ := estimateCleanupBytes(context.Background())
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
		defer cancel()

		steps := []string{
			"apt-get clean",
			"journalctl --vacuum-time=7d",
			"rm -rf /var/tmp/*",
		}

		for _, step := range steps {
			out, err := runPrivilegedShell(ctx, step)
			appendCleanupLog(fmt.Sprintf("$ %s", step))
			for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
				if strings.TrimSpace(line) != "" {
					appendCleanupLog(line)
				}
			}
			if err != nil {
				appendCleanupLog(fmt.Sprintf("step failed: %v", err))
			}
		}

		after, _ := estimateCleanupBytes(context.Background())
		freed := before - after
		if freed < 0 {
			freed = 0
		}

		cleanupMu.Lock()
		cleanupRunning = false
		cleanupDone = true
		cleanupLastAt = time.Now()
		cleanupLastFreed = freed
		cleanupLogs = append(cleanupLogs,
			fmt.Sprintf("[%s] Cleanup complete in %s", cleanupLastAt.Format(time.RFC3339), time.Since(start).Truncate(time.Second)),
			fmt.Sprintf("Estimated space freed: %s", humanBytes(uint64(freed))),
		)
		cleanupMu.Unlock()

		db.InsertAlert("maintenance", "info", "system", fmt.Sprintf("System cleanup completed (%s freed)", humanBytes(uint64(freed))), "", "system")
	}()

	writeJSON(w, http.StatusOK, map[string]any{"status": "running"})
}

func appendCleanupLog(line string) {
	cleanupMu.Lock()
	cleanupLogs = append(cleanupLogs, line)
	cleanupMu.Unlock()
}

func estimateCleanupBytes(ctx context.Context) (int64, error) {
	ctx2, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	out, err := runPrivilegedShell(ctx2, "du -sb /var/cache/apt/archives /var/log/journal /var/tmp 2>/dev/null | awk '{sum+=$1} END {print sum+0}'")
	if err != nil {
		return 0, err
	}
	n := strings.TrimSpace(out)
	v, convErr := strconv.ParseInt(n, 10, 64)
	if convErr != nil {
		return 0, convErr
	}
	return v, nil
}

func (h *Handlers) GetSecurityTools(w http.ResponseWriter, r *http.Request) {
	toolRunMu.Lock()
	out := make([]map[string]any, 0, len(securityToolCatalog))
	for _, t := range securityToolCatalog {
		st := toolRunStore[t.Name]
		out = append(out, map[string]any{
			"name":        t.Name,
			"label":       t.Label,
			"description": t.Description,
			"running":     st.Running,
			"last_status": st.LastStatus,
			"last_run_at": st.LastRunAt,
			"last_error":  st.LastError,
		})
	}
	toolRunMu.Unlock()
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) GetSecurityToolLogs(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(chi.URLParam(r, "name"))
	toolRunMu.Lock()
	st, ok := toolRunStore[name]
	if !ok {
		toolRunMu.Unlock()
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}
	resp := map[string]any{
		"name":        name,
		"running":     st.Running,
		"last_status": st.LastStatus,
		"last_run_at": st.LastRunAt,
		"last_error":  st.LastError,
		"logs":        append([]string{}, st.Logs...),
	}
	toolRunMu.Unlock()
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handlers) RunSecurityTool(w http.ResponseWriter, r *http.Request) {
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}
	name := strings.TrimSpace(chi.URLParam(r, "name"))
	tool, ok := getSecurityToolDef(name)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}

	if !commandExists(toolCommandBinary(tool.Name)) {
		writeError(w, http.StatusUnprocessableEntity, fmt.Sprintf("%s is not installed", tool.Label))
		return
	}

	toolRunMu.Lock()
	st := toolRunStore[name]
	if st.Running {
		toolRunMu.Unlock()
		writeError(w, http.StatusConflict, "tool already running")
		return
	}
	st.Running = true
	st.LastStatus = "running"
	st.LastError = ""
	st.Logs = []string{fmt.Sprintf("[%s] Starting %s", time.Now().Format(time.RFC3339), tool.Label)}
	toolRunMu.Unlock()

	go runSecurityToolJob(tool)
	writeJSON(w, http.StatusOK, map[string]any{"status": "running", "tool": name})
}

func (h *Handlers) InstallSecurityTool(w http.ResponseWriter, r *http.Request) {
	if !sudoAvailable() {
		writeError(w, http.StatusServiceUnavailable, sudoNotConfiguredMsg)
		return
	}
	name := strings.TrimSpace(chi.URLParam(r, "name"))
	_, ok := getSecurityToolDef(name)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown tool")
		return
	}

	pkg, ok := getSecurityToolPackage(name)
	if !ok {
		writeError(w, http.StatusBadRequest, "installation is not supported for this tool")
		return
	}

	if commandExists(toolCommandBinary(name)) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_installed"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	var output string
	var err error
	switch pkgFamily {
	case "apt":
		output, err = runPrivileged(ctx, "apt-get", "-y", "install", pkg)
	case "dnf", "yum":
		output, err = runPrivileged(ctx, pkgFamily, "-y", "install", pkg)
	case "pacman":
		output, err = runPrivileged(ctx, "pacman", "-S", "--noconfirm", pkg)
	case "zypper":
		output, err = runPrivileged(ctx, "zypper", "--non-interactive", "install", pkg)
	default:
		writeError(w, http.StatusServiceUnavailable, "unsupported package manager")
		return
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("package install failed: %v", err))
		return
	}

	invalidateManagedServicesCache()
	writeJSON(w, http.StatusOK, map[string]any{"status": "installed", "output": strings.TrimSpace(output)})
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func toolCommandBinary(name string) string {
	switch name {
	case "clamav":
		return "clamscan"
	default:
		return name
	}
}

func getSecurityToolPackage(name string) (string, bool) {
	switch name {
	case "chkrootkit":
		return "chkrootkit", true
	case "clamav":
		return "clamav", true
	case "rkhunter":
		return "rkhunter", true
	case "lynis":
		return "lynis", true
	default:
		return "", false
	}
}

func runSecurityToolJob(tool securityToolDef) {
	ctx, cancel := context.WithTimeout(context.Background(), tool.Timeout)
	defer cancel()

	var cmd *exec.Cmd
	if isRootUser() {
		cmd = exec.CommandContext(ctx, "bash", "-lc", tool.Command)
	} else {
		cmd = exec.CommandContext(ctx, "sudo", "-n", "bash", "-lc", tool.Command)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		finishToolRun(tool.Name, "failed", err.Error())
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		finishToolRun(tool.Name, "failed", err.Error())
		return
	}

	if err := cmd.Start(); err != nil {
		finishToolRun(tool.Name, "failed", err.Error())
		return
	}

	readPipe := func(sc *bufio.Scanner) {
		for sc.Scan() {
			appendToolLog(tool.Name, sc.Text())
		}
	}

	go readPipe(bufio.NewScanner(stdout))
	go readPipe(bufio.NewScanner(stderr))

	err = cmd.Wait()
	if ctx.Err() == context.DeadlineExceeded {
		finishToolRun(tool.Name, "failed", "tool run timed out")
		return
	}
	if err != nil {
		finishToolRun(tool.Name, "failed", err.Error())
		return
	}
	finishToolRun(tool.Name, "success", "")
}

func appendToolLog(name, line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	toolRunMu.Lock()
	if st, ok := toolRunStore[name]; ok {
		st.Logs = append(st.Logs, line)
	}
	toolRunMu.Unlock()
}

func finishToolRun(name, status, errText string) {
	toolRunMu.Lock()
	if st, ok := toolRunStore[name]; ok {
		st.Running = false
		st.LastStatus = status
		st.LastRunAt = time.Now().Format(time.RFC3339)
		st.LastError = errText
		if errText != "" {
			st.Logs = append(st.Logs, "ERROR: "+errText)
		} else {
			st.Logs = append(st.Logs, "Run completed successfully")
		}
	}
	toolRunMu.Unlock()
}

func getSecurityToolDef(name string) (securityToolDef, bool) {
	for _, t := range securityToolCatalog {
		if t.Name == name {
			return t, true
		}
	}
	return securityToolDef{}, false
}

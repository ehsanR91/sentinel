package api

import (
	"bufio"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/ehsanR91/sentinelcore/internal/db"
)

func (h *Handlers) GetAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			limit = v
		}
	}
	alerts, err := db.GetAlerts(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	writeJSON(w, http.StatusOK, alerts)
}

func (h *Handlers) MarkAlertRead(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	db.MarkAlertRead(id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) MarkAlertsRead(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if len(req.IDs) == 0 {
		writeError(w, http.StatusBadRequest, "no ids provided")
		return
	}
	for _, id := range req.IDs {
		db.MarkAlertRead(id)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	limit := 200
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			limit = v
		}
	}

	rows, err := db.DB().Query(`
		SELECT id, username, ip, success, reason, user_agent, ts
		FROM login_attempts
		ORDER BY ts DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	defer rows.Close()

	type auditRow struct {
		ID        int64  `json:"id"`
		Username  string `json:"username"`
		IP        string `json:"ip"`
		Success   bool   `json:"success"`
		Reason    string `json:"reason"`
		UserAgent string `json:"user_agent"`
		Ts        int64  `json:"ts"`
	}
	var out []auditRow
	for rows.Next() {
		var a auditRow
		var succ int
		if err := rows.Scan(&a.ID, &a.Username, &a.IP, &succ, &a.Reason, &a.UserAgent, &a.Ts); err == nil {
			a.Success = succ == 1
			out = append(out, a)
		}
	}
	if out == nil {
		out = []auditRow{}
	}
	writeJSON(w, http.StatusOK, out)
}

// TriggerAlert creates an alert from an internal event (called by other handlers).
func TriggerAlert(typ, severity, source, message, ip, username string) {
	db.InsertAlert(typ, severity, source, message, ip, username)
}

// ── System log ingestion ──────────────────────────────────────────────────────

// IngestSystemAlerts reads system logs and inserts new alerts into the DB.
// Called on startup and every 5 minutes from main.
func IngestSystemAlerts() {
	ingestFail2Ban()
	ingestPSAD()
	ingestAuthLog()
}

var reIP = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)

// ingestFail2Ban parses /var/log/fail2ban.log for Ban/Unban events.
func ingestFail2Ban() {
	lines := readLastLines("/var/log/fail2ban.log", 300)
	for _, line := range lines {
		if strings.Contains(line, " Ban ") {
			ip := reIP.FindString(line)
			jail := extractBracketWord(line)
			msg := "IP banned by fail2ban"
			if jail != "" {
				msg += " [" + jail + "]"
			}
			if ip != "" {
				msg += ": " + ip
			}
			db.InsertAlertDedup("ban", "warning", "fail2ban", msg, ip, "")
		} else if strings.Contains(line, "NOTICE") && strings.Contains(line, "restored") {
			// fail2ban service restored after downtime — info level
			db.InsertAlertDedup("service", "info", "fail2ban", strings.TrimSpace(line[clamp(strings.Index(line, "NOTICE"), 0, len(line)):]), "", "")
		}
	}
}

// ingestPSAD parses psad alert files and syslog for port-scan events.
func ingestPSAD() {
	// psad may write to its own log or to syslog
	for _, path := range []string{"/var/log/psad/alert", "/var/log/syslog", "/var/log/messages"} {
		lines := readLastLines(path, 500)
		for _, line := range lines {
			low := strings.ToLower(line)
			if !strings.Contains(low, "psad") {
				continue
			}
			if strings.Contains(low, "scan") || strings.Contains(low, "alert") || strings.Contains(low, "fw1drop") {
				ip := reIP.FindString(line)
				msg := strings.TrimSpace(line)
				if len(msg) > 200 {
					msg = msg[:200]
				}
				db.InsertAlertDedup("scan", "warning", "psad", msg, ip, "")
			}
		}
	}
}

// ingestAuthLog parses /var/log/auth.log for SSH failures and accepted logins.
func ingestAuthLog() {
	lines := readLastLines("/var/log/auth.log", 500)
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	for _, line := range lines {
		ts := parseSyslogTS(line)
		if ts > 0 && ts < cutoff {
			continue // skip entries older than 24h
		}
		low := strings.ToLower(line)
		ip := reIP.FindString(line)
		switch {
		case strings.Contains(low, "failed password") || strings.Contains(low, "authentication failure"):
			user := extractSSHUser(line)
			msg := "SSH failed login"
			if user != "" {
				msg += " for " + user
			}
			if ip != "" {
				msg += " from " + ip
			}
			db.InsertAlertDedup("auth_failure", "warning", "sshd", msg, ip, user)
		case strings.Contains(low, "invalid user"):
			user := extractSSHUser(line)
			msg := "SSH invalid user"
			if user != "" {
				msg += " " + user
			}
			if ip != "" {
				msg += " from " + ip
			}
			db.InsertAlertDedup("auth_failure", "warning", "sshd", msg, ip, user)
		}
	}
}

// readLastLines reads up to n lines from the tail of a file.
func readLastLines(path string, n int) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	ring := make([]string, n)
	idx, total := 0, 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 512*1024), 512*1024)
	for sc.Scan() {
		ring[idx%n] = sc.Text()
		idx++
		total++
	}
	if total == 0 {
		return nil
	}
	start := idx % n
	out := make([]string, 0, n)
	for i := 0; i < n && i < total; i++ {
		if l := ring[(start+i)%n]; l != "" {
			out = append(out, l)
		}
	}
	return out
}

// extractBracketWord extracts the first [word] from a log line (e.g. jail name).
func extractBracketWord(line string) string {
	s := strings.Index(line, "[")
	e := strings.Index(line, "]")
	if s >= 0 && e > s {
		return line[s+1 : e]
	}
	return ""
}

// extractSSHUser extracts the username from sshd log lines.
func extractSSHUser(line string) string {
	for _, marker := range []string{"for invalid user ", "for user ", "for "} {
		if idx := strings.Index(strings.ToLower(line), marker); idx >= 0 {
			rest := line[idx+len(marker):]
			if f := strings.Fields(rest); len(f) > 0 {
				return f[0]
			}
		}
	}
	return ""
}

// parseSyslogTS parses the leading timestamp from a syslog line ("Jan  2 15:04:05").
func parseSyslogTS(line string) int64 {
	if len(line) < 15 {
		return 0
	}
	t, err := time.Parse("Jan  2 15:04:05", line[:15])
	if err != nil {
		t, err = time.Parse("Jan 2 15:04:05", line[:14])
		if err != nil {
			return 0
		}
	}
	now := time.Now()
	t = t.AddDate(now.Year(), 0, 0)
	if t.After(now) {
		t = t.AddDate(-1, 0, 0)
	}
	return t.Unix()
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// GetDashboardLoginAttempts returns recent login attempts for the dashboard widget.
func (h *Handlers) GetDashboardLoginAttempts(w http.ResponseWriter, r *http.Request) {
	limit := 10
	rows, err := db.DB().Query(`
		SELECT username, ip, success, reason, ts
		FROM login_attempts
		ORDER BY ts DESC
		LIMIT ?`, limit,
	)
	if err != nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	defer rows.Close()

	type attempt struct {
		Username string `json:"username"`
		IP       string `json:"ip"`
		Success  bool   `json:"success"`
		Reason   string `json:"reason"`
		Ts       int64  `json:"ts"`
	}
	var out []attempt
	for rows.Next() {
		var a attempt
		var succ int
		if err := rows.Scan(&a.Username, &a.IP, &succ, &a.Reason, &a.Ts); err == nil {
			a.Success = succ == 1
			out = append(out, a)
		}
	}
	if out == nil {
		out = []attempt{}
	}
	writeJSON(w, http.StatusOK, out)
}

// GetDashboardLayout returns the saved dashboard layout for the current user
func (h *Handlers) GetDashboardLayout(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	userIDFloat, ok := claims["uid"].(float64)
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{
			"widgets":    nil,
			"layoutMode": "flexible",
		})
		return
	}
	userID := int64(userIDFloat)

	row := db.DB().QueryRow(`
		SELECT widgets, layout_mode
		FROM dashboard_layout
		WHERE user_id = ?
	`, userID)

	var widgetsStr, layoutMode string
	if err := row.Scan(&widgetsStr, &layoutMode); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"widgets":    nil,
			"layoutMode": "flexible",
		})
		return
	}

	var widgets []map[string]any
	if widgetsStr != "" {
		if err := json.Unmarshal([]byte(widgetsStr), &widgets); err != nil {
			writeJSON(w, http.StatusOK, map[string]any{
				"widgets":    nil,
				"layoutMode": layoutMode,
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"widgets":    widgets,
		"layoutMode": layoutMode,
	})
}

// SaveDashboardLayout saves the dashboard layout for the current user
func (h *Handlers) SaveDashboardLayout(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	userIDFloat, ok := claims["uid"].(float64)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid user")
		return
	}
	userID := int64(userIDFloat)

	var req struct {
		Widgets    []map[string]any `json:"widgets"`
		LayoutMode string           `json:"layoutMode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	widgetsJSON, err := json.Marshal(req.Widgets)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encoding error")
		return
	}

	if req.LayoutMode == "" {
		req.LayoutMode = "flexible"
	}

	_, err = db.DB().Exec(`
		INSERT INTO dashboard_layout (user_id, widgets, layout_mode)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			widgets = excluded.widgets,
			layout_mode = excluded.layout_mode
	`, userID, string(widgetsJSON), req.LayoutMode)

	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

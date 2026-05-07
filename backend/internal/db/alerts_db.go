package db

// Alert mirrors the alerts table row.
type Alert struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Message  string `json:"message"`
	IP       string `json:"ip"`
	Username string `json:"username"`
	Read     bool   `json:"read"`
	Ts       int64  `json:"ts"`
}

// InsertAlert creates a new alert row.
func InsertAlert(typ, severity, source, message, ip, username string) error {
	_, err := db.Exec(
		`INSERT INTO alerts(type,severity,source,message,ip,username,read,ts)
		 VALUES(?,?,?,?,?,?,0,?)`,
		typ, severity, source, message, ip, username, unixNow(),
	)
	return err
}

// InsertAlertDedup inserts an alert only when no identical (source+message)
// alert already exists within the last hour, preventing log-ingestion spam.
func InsertAlertDedup(typ, severity, source, message, ip, username string) error {
	var count int
	db.QueryRow(
		`SELECT COUNT(*) FROM alerts WHERE source=? AND message=? AND ts >= ?`,
		source, message, unixNow()-3600,
	).Scan(&count)
	if count > 0 {
		return nil
	}
	return InsertAlert(typ, severity, source, message, ip, username)
}

// GetAlerts returns the most recent alerts, newest first.
func GetAlerts(limit int) ([]Alert, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := db.Query(
		`SELECT id,type,severity,source,message,ip,username,read,ts
		 FROM alerts ORDER BY ts DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Alert
	for rows.Next() {
		var a Alert
		var read int
		if err := rows.Scan(&a.ID, &a.Type, &a.Severity, &a.Source,
			&a.Message, &a.IP, &a.Username, &read, &a.Ts); err == nil {
			a.Read = read == 1
			out = append(out, a)
		}
	}
	if out == nil {
		out = []Alert{}
	}
	return out, nil
}

// MarkAlertRead sets read=1 for the given alert id.
func MarkAlertRead(id int64) error {
	_, err := db.Exec(`UPDATE alerts SET read=1 WHERE id=?`, id)
	return err
}

// UnreadAlertCount returns the count of unread alerts.
func UnreadAlertCount() (int, error) {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM alerts WHERE read=0`).Scan(&n)
	return n, err
}

// ManualBan represents an admin-added IP ban.
type ManualBan struct {
	IP       string `json:"ip"`
	Reason   string `json:"reason"`
	BannedBy string `json:"banned_by"`
	Ts       int64  `json:"ts"`
}

// AddManualBan inserts or replaces an IP ban.
func AddManualBan(ip, reason, bannedBy string) error {
	_, err := db.Exec(
		`INSERT INTO manual_bans(ip,reason,banned_by,ts) VALUES(?,?,?,?)
		 ON CONFLICT(ip) DO UPDATE SET reason=excluded.reason, banned_by=excluded.banned_by, ts=excluded.ts`,
		ip, reason, bannedBy, unixNow(),
	)
	return err
}

// RemoveManualBan deletes an IP ban.
func RemoveManualBan(ip string) error {
	_, err := db.Exec(`DELETE FROM manual_bans WHERE ip=?`, ip)
	return err
}

// GetManualBans returns all manual bans.
func GetManualBans() ([]ManualBan, error) {
	rows, err := db.Query(`SELECT ip,reason,banned_by,ts FROM manual_bans ORDER BY ts DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ManualBan
	for rows.Next() {
		var b ManualBan
		if err := rows.Scan(&b.IP, &b.Reason, &b.BannedBy, &b.Ts); err == nil {
			out = append(out, b)
		}
	}
	if out == nil {
		out = []ManualBan{}
	}
	return out, nil
}

// GetBruteForceBans returns IPs that have exceeded the failure threshold
// within the given window (minutes), combining with manual bans.
func GetBruteForceBans(threshold, windowMinutes int) ([]map[string]any, error) {
	rows, err := db.Query(`
		SELECT ip, COUNT(*) as attempts, MAX(ts) as last_seen
		FROM login_attempts
		WHERE success=0 AND ts >= ?
		GROUP BY ip
		HAVING attempts >= ?
		ORDER BY last_seen DESC`,
		unixNow()-int64(windowMinutes*60), threshold,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var ip string
		var attempts int
		var lastSeen int64
		if err := rows.Scan(&ip, &attempts, &lastSeen); err == nil {
			out = append(out, map[string]any{
				"ip":        ip,
				"attempts":  attempts,
				"last_seen": lastSeen,
				"source":    "brute_force",
			})
		}
	}
	if out == nil {
		out = []map[string]any{}
	}
	return out, nil
}

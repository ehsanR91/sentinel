package db

import "time"

type AuditEvent struct {
	ID       int64
	Username string
	IP       string
	Action   string
	Target   string
	Details  string
	Success  bool
	Ts       int64
}

func InsertAuditEvent(username, ip, action, target, details string, success bool) error {
	succeeded := 0
	if success {
		succeeded = 1
	}
	_, err := db.Exec(
		`INSERT INTO audit_events(username, ip, action, target, details, success, ts)
		 VALUES(?, ?, ?, ?, ?, ?, ?)`,
		username, ip, action, target, details, succeeded, time.Now().Unix(),
	)
	return err
}

func ListAuditEvents(limit int) ([]AuditEvent, error) {
	rows, err := db.Query(
		`SELECT id, username, ip, action, target, details, success, ts
		 FROM audit_events ORDER BY ts DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]AuditEvent, 0, limit)
	for rows.Next() {
		var entry AuditEvent
		var succeeded int
		if err := rows.Scan(&entry.ID, &entry.Username, &entry.IP, &entry.Action, &entry.Target, &entry.Details, &succeeded, &entry.Ts); err != nil {
			return nil, err
		}
		entry.Success = succeeded == 1
		out = append(out, entry)
	}
	if out == nil {
		out = []AuditEvent{}
	}
	return out, rows.Err()
}

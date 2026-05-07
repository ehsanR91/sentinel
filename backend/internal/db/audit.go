package db

import "time"

// LogLoginAttempt records a login attempt in the database.
func LogLoginAttempt(username, ip, reason string, success bool, userAgent string) error {
	s := 0
	if success {
		s = 1
	}
	_, err := db.Exec(
		`INSERT INTO login_attempts(username, ip, success, reason, user_agent, ts)
		 VALUES(?, ?, ?, ?, ?, ?)`,
		username, ip, s, reason, userAgent, time.Now().Unix(),
	)
	return err
}

// TotalRecentFailed returns total failed login attempts across all IPs
// in the last windowMinutes minutes.
func TotalRecentFailed(windowMinutes int) (int, error) {
	since := unixNow() - int64(windowMinutes*60)
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM login_attempts WHERE success=0 AND ts>=?`, since,
	).Scan(&count)
	return count, err
}

// in the last windowMinutes minutes.
func RecentFailedCount(ip string, windowMinutes int) (int, error) {
	since := time.Now().Add(-time.Duration(windowMinutes) * time.Minute).Unix()
	var count int
	err := db.QueryRow(
		`SELECT COUNT(*) FROM login_attempts WHERE ip=? AND success=0 AND ts>=?`,
		ip, since,
	).Scan(&count)
	return count, err
}

package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	_ "modernc.org/sqlite"
)

var (
	db   *sql.DB
	once sync.Once

	settingsCacheMu sync.RWMutex
	settingsCache   = map[string]settingCacheEntry{}
)

type settingCacheEntry struct {
	value  string
	exists bool
}

// DB returns the shared database instance (panics if Init not called).
func DB() *sql.DB { return db }

func resetSettingsCache() {
	settingsCacheMu.Lock()
	settingsCache = map[string]settingCacheEntry{}
	settingsCacheMu.Unlock()
}

// Init opens the SQLite database, creates parent dirs, and runs migrations.
func Init(path, secretsKeyPath string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
		return fmt.Errorf("db dir: %w", err)
	}
	if err := initSecrets(secretsKeyPath); err != nil {
		return fmt.Errorf("secrets init: %w", err)
	}
	conn, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return err
	}
	conn.SetMaxOpenConns(1)
	db = conn
	resetSettingsCache()
	return migrate()
}

func migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			username      TEXT    UNIQUE NOT NULL,
			password_hash TEXT    NOT NULL,
			role          TEXT    NOT NULL DEFAULT 'viewer',
			totp_secret   TEXT    NOT NULL DEFAULT '',
			totp_enabled  INTEGER NOT NULL DEFAULT 0,
			email         TEXT    NOT NULL DEFAULT '',
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS login_attempts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT    NOT NULL,
			ip         TEXT    NOT NULL,
			success    INTEGER NOT NULL,
			reason     TEXT    NOT NULL DEFAULT '',
			user_agent TEXT    NOT NULL DEFAULT '',
			ts         INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_login_ip_ts ON login_attempts(ip, ts)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			username   TEXT    NOT NULL DEFAULT '',
			ip         TEXT    NOT NULL DEFAULT '',
			action     TEXT    NOT NULL,
			target     TEXT    NOT NULL DEFAULT '',
			details    TEXT    NOT NULL DEFAULT '',
			success    INTEGER NOT NULL DEFAULT 1,
			ts         INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_ts ON audit_events(ts DESC)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS alerts (
			id       INTEGER PRIMARY KEY AUTOINCREMENT,
			type     TEXT    NOT NULL DEFAULT 'info',
			severity TEXT    NOT NULL DEFAULT 'info',
			source   TEXT    NOT NULL DEFAULT 'system',
			message  TEXT    NOT NULL,
			ip       TEXT    NOT NULL DEFAULT '',
			username TEXT    NOT NULL DEFAULT '',
			read     INTEGER NOT NULL DEFAULT 0,
			ts       INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_ts ON alerts(ts)`,
		`CREATE TABLE IF NOT EXISTS manual_bans (
			ip        TEXT    PRIMARY KEY,
			reason    TEXT    NOT NULL DEFAULT '',
			banned_by TEXT    NOT NULL DEFAULT '',
			ts        INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			name          TEXT    NOT NULL,
			description   TEXT    NOT NULL DEFAULT '',
			command       TEXT    NOT NULL,
			schedule_kind TEXT    NOT NULL DEFAULT 'manual',
			schedule_expr TEXT    NOT NULL DEFAULT '',
			enabled       INTEGER NOT NULL DEFAULT 1,
			created_by    TEXT    NOT NULL DEFAULT '',
			created_at    INTEGER NOT NULL,
			updated_at    INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_tasks_enabled ON tasks(enabled)`,
		`CREATE TABLE IF NOT EXISTS task_runs (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id    INTEGER NOT NULL,
			started_at INTEGER NOT NULL,
			ended_at   INTEGER NOT NULL DEFAULT 0,
			status     TEXT    NOT NULL DEFAULT 'running',
			triggered_by TEXT  NOT NULL DEFAULT 'system',
			output     TEXT    NOT NULL DEFAULT '',
			exit_code  INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY(task_id) REFERENCES tasks(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_task_runs_task_started ON task_runs(task_id, started_at DESC)`,
		`CREATE TABLE IF NOT EXISTS dashboard_layout (
			user_id     INTEGER PRIMARY KEY,
			widgets     TEXT    NOT NULL DEFAULT '[]',
			layout_mode TEXT    NOT NULL DEFAULT 'flexible',
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS user_lock_settings (
			user_id INTEGER PRIMARY KEY,
			enabled INTEGER NOT NULL DEFAULT 0,
			pin_hash TEXT NOT NULL DEFAULT '',
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	if err := ensureColumn("dashboard_layout", "state_json", `ALTER TABLE dashboard_layout ADD COLUMN state_json TEXT NOT NULL DEFAULT '{}'`); err != nil {
		return fmt.Errorf("migrate dashboard layout state_json: %w", err)
	}
	return nil
}

func ensureColumn(table, column, alterSQL string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			return err
		}
		if strings.EqualFold(name, column) {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(alterSQL)
	return err
}

// SeedAdmin creates the initial admin user if the users table is empty.
func SeedAdmin(username, passwordHash string) (bool, error) {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
		return false, err
	}
	if count > 0 {
		return false, nil
	}
	now := unixNow()
	_, err := db.Exec(
		`INSERT INTO users (username, password_hash, role, created_at, updated_at)
		 VALUES (?, ?, 'superadmin', ?, ?)`,
		username, passwordHash, now, now,
	)
	if err != nil {
		return false, err
	}
	return true, nil
}

// CloseAndReplace closes the current DB connection, replaces the DB file
// with srcPath, then reopens. Used for importing a backup.
func CloseAndReplace(srcPath, destPath string) error {
	if err := db.Close(); err != nil {
		return fmt.Errorf("close db: %w", err)
	}
	db = nil

	if err := os.Rename(srcPath, destPath); err != nil {
		// Rename across filesystems fails; fall back to copy
		if err2 := copyFile(srcPath, destPath); err2 != nil {
			return fmt.Errorf("replace db: %w", err2)
		}
	}

	conn, err := sql.Open("sqlite", destPath+"?_journal_mode=WAL&_foreign_keys=on&_busy_timeout=5000")
	if err != nil {
		return fmt.Errorf("reopen db: %w", err)
	}
	conn.SetMaxOpenConns(1)
	db = conn
	resetSettingsCache()
	return migrate()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 1<<20)
	for {
		n, readErr := in.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
		}
		if readErr != nil {
			if readErr.Error() == "EOF" {
				break
			}
			return readErr
		}
	}
	return nil
}

func GetSetting(key, defaultVal string) string {
	settingsCacheMu.RLock()
	if entry, ok := settingsCache[key]; ok {
		settingsCacheMu.RUnlock()
		if !entry.exists {
			return defaultVal
		}
		return entry.value
	}
	settingsCacheMu.RUnlock()

	var val string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err != nil {
		settingsCacheMu.Lock()
		settingsCache[key] = settingCacheEntry{exists: false}
		settingsCacheMu.Unlock()
		return defaultVal
	}
	settingsCacheMu.Lock()
	settingsCache[key] = settingCacheEntry{value: val, exists: true}
	settingsCacheMu.Unlock()
	return val
}

// SetSetting upserts a key→value pair in the settings table.
func SetSetting(key, value string) error {
	_, err := db.Exec(
		`INSERT INTO settings(key,value) VALUES(?,?)
		 ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	if err == nil {
		settingsCacheMu.Lock()
		settingsCache[key] = settingCacheEntry{value: value, exists: true}
		settingsCacheMu.Unlock()
	}
	return err
}

// RotateSecretsKey re-encrypts all protected DB values with a freshly generated
// master key and atomically replaces the on-disk key file.
func RotateSecretsKey(keyPath string) error {
	keyPath = strings.TrimSpace(keyPath)
	if keyPath == "" {
		return fmt.Errorf("secrets key path is not configured")
	}

	oldKeys, err := loadOrCreateSecretKeys(keyPath)
	if err != nil {
		return fmt.Errorf("load current secrets keys: %w", err)
	}
	fi, err := os.Stat(keyPath)
	if err != nil {
		return fmt.Errorf("stat secrets key file: %w", err)
	}

	newKey, err := generateSecretKey()
	if err != nil {
		return fmt.Errorf("generate new secrets key: %w", err)
	}

	ringKeys := append([][]byte{newKey}, oldKeys...)
	if err := writeSecretKeysAtomic(keyPath, ringKeys, fi.Mode().Perm()); err != nil {
		return fmt.Errorf("write transitional secrets key ring: %w", err)
	}
	if err := initSecrets(keyPath); err != nil {
		return fmt.Errorf("reload transitional secrets key ring: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin rotation transaction: %w", err)
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		_ = writeSecretKeysAtomic(keyPath, oldKeys, fi.Mode().Perm())
		_ = initSecrets(keyPath)
		return cause
	}

	rows, err := tx.Query(`SELECT id, totp_secret FROM users WHERE totp_secret <> ''`)
	if err != nil {
		return rollback(fmt.Errorf("query totp secrets: %w", err))
	}
	for rows.Next() {
		var id int64
		var stored string
		if err := rows.Scan(&id, &stored); err != nil {
			rows.Close()
			return rollback(fmt.Errorf("scan totp secret: %w", err))
		}
		plain, err := decryptAtRest(stored)
		if err != nil {
			rows.Close()
			return rollback(fmt.Errorf("decrypt totp secret for user %d: %w", id, err))
		}
		reEnc, err := encryptAtRest(plain)
		if err != nil {
			rows.Close()
			return rollback(fmt.Errorf("re-encrypt totp secret for user %d: %w", id, err))
		}
		if _, err := tx.Exec(`UPDATE users SET totp_secret=?, updated_at=? WHERE id=?`, reEnc, unixNow(), id); err != nil {
			rows.Close()
			return rollback(fmt.Errorf("update totp secret for user %d: %w", id, err))
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return rollback(fmt.Errorf("iterate totp secrets: %w", err))
	}
	rows.Close()

	settingsRows, err := tx.Query(`SELECT key, value FROM settings WHERE value <> ''`)
	if err != nil {
		return rollback(fmt.Errorf("query settings secrets: %w", err))
	}
	for settingsRows.Next() {
		var key, stored string
		if err := settingsRows.Scan(&key, &stored); err != nil {
			settingsRows.Close()
			return rollback(fmt.Errorf("scan settings value: %w", err))
		}
		if key != "smtp_pass" && !strings.HasPrefix(stored, encPrefix) {
			continue
		}
		plain, err := decryptAtRest(stored)
		if err != nil {
			settingsRows.Close()
			return rollback(fmt.Errorf("decrypt setting %s: %w", key, err))
		}
		if plain == "" {
			continue
		}
		reEnc, err := encryptAtRest(plain)
		if err != nil {
			settingsRows.Close()
			return rollback(fmt.Errorf("re-encrypt setting %s: %w", key, err))
		}
		if _, err := tx.Exec(`UPDATE settings SET value=? WHERE key=?`, reEnc, key); err != nil {
			settingsRows.Close()
			return rollback(fmt.Errorf("update setting %s: %w", key, err))
		}
	}
	if err := settingsRows.Err(); err != nil {
		settingsRows.Close()
		return rollback(fmt.Errorf("iterate settings values: %w", err))
	}
	settingsRows.Close()

	if err := tx.Commit(); err != nil {
		return rollback(fmt.Errorf("commit secrets rotation: %w", err))
	}
	resetSettingsCache()

	if err := SetSetting("last_master_key_rotation", fmt.Sprintf("%d", unixNow())); err != nil {
		return fmt.Errorf("record last master key rotation: %w", err)
	}

	if err := writeSecretKeysAtomic(keyPath, [][]byte{newKey}, fi.Mode().Perm()); err != nil {
		_ = initSecrets(keyPath)
		return fmt.Errorf("compact rotated secrets key file: %w", err)
	}
	if err := initSecrets(keyPath); err != nil {
		return fmt.Errorf("reload rotated secrets key: %w", err)
	}
	return nil
}

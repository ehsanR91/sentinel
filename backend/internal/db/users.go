package db

import (
	"database/sql"
	"errors"
	"time"
)

// User mirrors the users table.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Role         string
	TOTPSecret   string
	TOTPEnabled  bool
	Email        string
	CreatedAt    int64
	UpdatedAt    int64
}

// ErrNotFound is returned when a user lookup yields no row.
var ErrNotFound = errors.New("user not found")

func unixNow() int64 { return time.Now().Unix() }

// GetUserByUsername fetches a user by their username.
func GetUserByUsername(username string) (*User, error) {
	row := db.QueryRow(
		`SELECT id, username, password_hash, role, totp_secret, totp_enabled, email, created_at, updated_at
		 FROM users WHERE username = ?`, username,
	)
	return scanUser(row)
}

// GetUser fetches a user by their primary key.
func GetUser(id int64) (*User, error) {
	row := db.QueryRow(
		`SELECT id, username, password_hash, role, totp_secret, totp_enabled, email, created_at, updated_at
		 FROM users WHERE id = ?`, id,
	)
	return scanUser(row)
}

// UpdateTOTP saves a TOTP secret and enabled flag for the given user.
func UpdateTOTP(userID int64, secret string, enabled bool) error {
	en := 0
	if enabled {
		en = 1
	}
	storedSecret := secret
	if secret != "" {
		enc, err := encryptAtRest(secret)
		if err != nil {
			return err
		}
		storedSecret = enc
	}
	_, err := db.Exec(
		`UPDATE users SET totp_secret=?, totp_enabled=?, updated_at=? WHERE id=?`,
		storedSecret, en, unixNow(), userID,
	)
	return err
}

// CreateUser inserts a new user.
func CreateUser(username, passwordHash, role, email string) error {
	now := unixNow()
	_, err := db.Exec(
		`INSERT INTO users(username,password_hash,role,email,created_at,updated_at)
		 VALUES(?,?,?,?,?,?)`,
		username, passwordHash, role, email, now, now,
	)
	return err
}

// UpdateUser changes role and/or email for a user by id.
func UpdateUser(id int64, role, email string) error {
	_, err := db.Exec(
		`UPDATE users SET role=?, email=?, updated_at=? WHERE id=?`,
		role, email, unixNow(), id,
	)
	return err
}

// DeleteUser removes a user by id. Returns ErrNotFound if no row was deleted.
func DeleteUser(id int64) error {
	res, err := db.Exec(`DELETE FROM users WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListUsers returns all users ordered by id.
func ListUsers() ([]User, error) {
	rows, err := db.Query(
		`SELECT id,username,password_hash,role,totp_secret,totp_enabled,email,created_at,updated_at
		 FROM users ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		u := User{}
		var totpEnabled int
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role,
			&u.TOTPSecret, &totpEnabled, &u.Email, &u.CreatedAt, &u.UpdatedAt); err == nil {
			dec, decErr := decryptAtRest(u.TOTPSecret)
			if decErr != nil {
				return nil, decErr
			}
			u.TOTPSecret = dec
			u.TOTPEnabled = totpEnabled == 1
			out = append(out, u)
		}
	}
	if out == nil {
		out = []User{}
	}
	return out, nil
}

// UpdatePassword sets a new argon2id password hash for the given username.
func UpdatePassword(username, hash string) error {
	res, err := db.Exec(`UPDATE users SET password_hash=?, updated_at=? WHERE username=?`, hash, unixNow(), username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResetTOTP disables TOTP for the given username.
func ResetTOTP(username string) error {
	res, err := db.Exec(`UPDATE users SET totp_secret='', totp_enabled=0, updated_at=? WHERE username=?`, unixNow(), username)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ListUsernames returns all usernames in the users table.
func ListUsernames() ([]string, error) {
	rows, err := db.Query(`SELECT username FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			names = append(names, name)
		}
	}
	return names, nil
}

func scanUser(row *sql.Row) (*User, error) {
	u := &User{}
	var totpEnabled int
	err := row.Scan(
		&u.ID, &u.Username, &u.PasswordHash, &u.Role,
		&u.TOTPSecret, &totpEnabled, &u.Email, &u.CreatedAt, &u.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	dec, decErr := decryptAtRest(u.TOTPSecret)
	if decErr != nil {
		return nil, decErr
	}
	u.TOTPSecret = dec
	u.TOTPEnabled = totpEnabled == 1
	return u, nil
}

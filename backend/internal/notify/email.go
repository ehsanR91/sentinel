package notify

import (
	"fmt"
	"net/smtp"
	"strings"
	"time"
)

// Mailer sends alert emails via SMTP.
type Mailer struct {
	Host       string
	Port       string
	User       string
	Password   string
	AlertEmail string
}

// NewMailer constructs a Mailer. Returns a no-op mailer if Host is empty.
func NewMailer(host, port, user, pass, alertEmail string) *Mailer {
	return &Mailer{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   pass,
		AlertEmail: alertEmail,
	}
}

// Send dispatches an email. Returns nil immediately if SMTP is not configured.
func (m *Mailer) Send(subject, body string) error {
	if m.Host == "" || m.AlertEmail == "" {
		return nil
	}
	addr := m.Host + ":" + m.Port
	auth := smtp.PlainAuth("", m.User, m.Password, m.Host)
	msg := strings.Join([]string{
		"From: SentinelCore <" + m.User + ">",
		"To: " + m.AlertEmail,
		"Subject: [SentinelCore] " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		body,
	}, "\r\n")
	return smtp.SendMail(addr, auth, m.User, []string{m.AlertEmail}, []byte(msg))
}

func (m *Mailer) AlertFailedLogin(username, ip string) {
	ts := time.Now().Format("2006-01-02 15:04:05 UTC")
	body := fmt.Sprintf(
		"Failed login attempt\n\nUsername: %s\nIP: %s\nTime: %s\n\nThis is an automated security alert from SentinelCore.",
		username, ip, ts,
	)
	go m.Send("Failed Login Attempt — "+ip, body) //nolint
}

func (m *Mailer) Alert2FAFailure(username, ip string) {
	ts := time.Now().Format("2006-01-02 15:04:05 UTC")
	body := fmt.Sprintf(
		"TOTP 2FA verification failed\n\nUsername: %s\nIP: %s\nTime: %s\n\nThis is an automated security alert from SentinelCore.",
		username, ip, ts,
	)
	go m.Send("2FA Verification Failure — "+ip, body) //nolint
}

func (m *Mailer) AlertBruteForce(ip string) {
	ts := time.Now().Format("2006-01-02 15:04:05 UTC")
	body := fmt.Sprintf(
		"Brute-force login attempt detected\n\nIP: %s\nTime: %s\n\nMultiple failed login attempts were detected from this IP address. The request has been rejected.",
		ip, ts,
	)
	go m.Send("Brute-Force Detected — "+ip, body) //nolint
}

package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	appauth "github.com/ehsanR91/sentinelcore/internal/auth"
	"github.com/ehsanR91/sentinelcore/internal/db"
)

var termUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // allow non-browser clients (curl, native apps)
		}
		return strings.HasPrefix(origin, "http://"+r.Host) ||
			strings.HasPrefix(origin, "https://"+r.Host)
	},
}

type termMsg struct {
	Type     string `json:"type"`
	Data     string `json:"data,omitempty"`
	TOTPCode string `json:"totp_code,omitempty"`
	Cmd      string `json:"cmd,omitempty"`
}

// termSession holds per-connection elevation state.
type termSession struct {
	mu            sync.Mutex
	writeMu       sync.Mutex
	elevated      bool
	elevatedUntil time.Time
	username      string
	ip            string
	activeStdin   io.WriteCloser
	activeCancel  context.CancelFunc
}

func sendTermMsg(conn *websocket.Conn, sess *termSession, msg termMsg) {
	sess.writeMu.Lock()
	defer sess.writeMu.Unlock()
	_ = conn.WriteJSON(msg)
}

func (s *termSession) startActiveCommand(stdin io.WriteCloser, cancel context.CancelFunc) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeStdin != nil {
		return false
	}
	s.activeStdin = stdin
	s.activeCancel = cancel
	return true
}

func (s *termSession) clearActiveCommand() {
	s.mu.Lock()
	stdin := s.activeStdin
	cancel := s.activeCancel
	s.activeStdin = nil
	s.activeCancel = nil
	s.mu.Unlock()
	if stdin != nil {
		_ = stdin.Close()
	}
	if cancel != nil {
		cancel()
	}
}

func (s *termSession) stopActiveCommand() {
	s.clearActiveCommand()
}

func (s *termSession) sendToActiveCommandInput(input string) (bool, error) {
	s.mu.Lock()
	stdin := s.activeStdin
	s.mu.Unlock()
	if stdin == nil {
		return false, nil
	}
	_, err := io.WriteString(stdin, input+"\n")
	return true, err
}

// When backend runs as root, commands are executed as this user.
var terminalRunAsUser = defaultTerminalRunAsUser()

func (s *termSession) isElevated() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.elevated && time.Now().Before(s.elevatedUntil)
}

func (s *termSession) elevate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.elevated = true
	s.elevatedUntil = time.Now().Add(5 * time.Minute)
}

func (s *termSession) revoke() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.elevated = false
}

func defaultTerminalRunAsUser() string {
	u := strings.TrimSpace(os.Getenv("TERMINAL_RUN_AS_USER"))
	if u == "" {
		return "deploy"
	}
	return u
}

// ─── Command risk classification ─────────────────────────────────────────────

// These patterns are NEVER allowed — catastrophic destructive potential.
var blockedPatterns = []string{
	"rm -rf /",
	"rm -rf /*",
	"rm --no-preserve-root",
	"dd if=/dev/zero of=/dev/sd",
	"dd if=/dev/random of=/dev/sd",
	"dd if=/dev/null of=/dev/sd",
	"mkfs",
	"> /dev/sd",
	"chmod -r 000 /",
	"chmod -r 777 /",
	"chown -r root /",
	":(){ :|:& };:", // fork bomb
	"fork bomb",
}

// These patterns require 2FA elevation to execute.
var highRiskPatterns = []string{
	"rm -rf",
	"rm -r ",
	"rm -f ",
	"shutdown",
	"reboot",
	"halt",
	"poweroff",
	"init 0",
	"init 6",
	"kill -9",
	"kill -",
	"killall",
	"pkill",
	"userdel",
	"useradd",
	"usermod",
	"passwd",
	"chpasswd",
	"visudo",
	"apt remove",
	"apt purge",
	"apt autoremove",
	"dpkg --remove",
	"dpkg --purge",
	"iptables -f",
	"iptables --flush",
	"ufw --force reset",
	"ufw reset",
	"ufw disable",
	"crontab -r",
	"truncate ",
	"shred ",
	"wipe ",
	"> /etc/",
	">> /etc/",
	"tee /etc/",
	"|bash",
	"| bash",
	"|sh",
	"| sh",
	"chmod 777",
	"chmod 000",
	"sudo su",
	"sudo -i",
	"sudo -s",
	"systemctl stop sentinelcore",
	"systemctl disable sentinelcore",
	"systemctl mask",
	"journalctl --vacuum",
}

// These commands are allowed in non-elevated mode.
var safeCommandPrefixes = []string{
	"help",
	"pwd",
	"whoami",
	"id",
	"date",
	"uptime",
	"hostname",
	"uname",
	"ls",
	"df",
	"free",
	"ps",
	"top",
	"ss",
	"ip",
	"docker ps",
	"docker stats --no-stream",
	"systemctl status",
	"systemctl is-active",
	"journalctl -u",
	"ufw status",
	"fail2ban-client status",
}

var riskyShellOperators = []string{"&&", "||", ";", "|", ">", "<", "`", "$()"}

type cmdRisk int

const (
	riskNormal cmdRisk = iota
	riskHigh
	riskBlocked
)

func classifyCommand(cmd string) (cmdRisk, string) {
	lower := strings.ToLower(strings.TrimSpace(cmd))
	if lower == "" {
		return riskNormal, ""
	}

	for _, pat := range blockedPatterns {
		if strings.Contains(lower, strings.ToLower(pat)) {
			return riskBlocked, pat
		}
	}

	for _, op := range riskyShellOperators {
		if strings.Contains(lower, op) {
			return riskHigh, op
		}
	}

	if isSafeCommand(lower) {
		return riskNormal, ""
	}

	for _, pat := range highRiskPatterns {
		if strings.Contains(lower, strings.ToLower(pat)) {
			return riskHigh, pat
		}
	}

	// Default-deny for non-allowlisted commands unless session is elevated.
	return riskHigh, "not in non-elevated allowlist"
}

func isSafeCommand(lowerCmd string) bool {
	for _, p := range safeCommandPrefixes {
		if lowerCmd == p || strings.HasPrefix(lowerCmd, p+" ") {
			return true
		}
	}
	return false
}

func effectiveExecUser() string {
	if os.Geteuid() == 0 && terminalRunAsUser != "" && terminalRunAsUser != "root" {
		return terminalRunAsUser
	}
	cur, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return cur.Username
}

func buildExecCommand(ctx context.Context, cmd string) *exec.Cmd {
	if os.Geteuid() == 0 && terminalRunAsUser != "" && terminalRunAsUser != "root" {
		return exec.CommandContext(ctx, "runuser", "-u", terminalRunAsUser, "--", "bash", "-lc", cmd)
	}
	return exec.CommandContext(ctx, "bash", "-lc", cmd)
}

// ─── WebSocket handler ───────────────────────────────────────────────────────

func (h *Handlers) TerminalWS(w http.ResponseWriter, r *http.Request) {
	conn, err := termUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)

	sess := &termSession{
		username: username,
		ip:       clientIP(r),
	}
	defer sess.stopActiveCommand()

	hostname, _ := os.Hostname()
	execUser := effectiveExecUser()
	sendTermMsg(conn, sess, termMsg{
		Type: "output",
		Data: "\x1b[1;32m[SentinelCore Terminal]\x1b[0m Authenticated as \x1b[1;33m" + username + "\x1b[0m\r\n" +
			"\x1b[0;36mCommand execution user: \x1b[1;33m" + execUser + "\x1b[0m\r\n" +
			"\x1b[0;36mAll commands are audited. High-risk commands require 2FA elevation.\x1b[0m\r\n" +
			"\x1b[2mNon-elevated mode only allows a small safe command set. Type 'help'.\x1b[0m\r\n",
	})
	// Send hostname so frontend can display it
	sendTermMsg(conn, sess, termMsg{Type: "info", Data: hostname})

	db.LogLoginAttempt(username, sess.ip, "terminal_connect", true, "ws")

	for {
		_, rawMsg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg termMsg
		if err := json.Unmarshal(rawMsg, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "unlock":
			h.handleTerminalUnlock(conn, sess, msg.TOTPCode)
		case "revoke":
			sess.revoke()
			sendTermMsg(conn, sess, termMsg{Type: "revoked", Data: "High-risk mode disabled."})
		case "input":
			h.handleTerminalInput(conn, sess, msg.Data)
		}
	}
}

func (h *Handlers) handleTerminalInput(conn *websocket.Conn, sess *termSession, input string) {
	if forwarded, err := sess.sendToActiveCommandInput(input); forwarded {
		if err != nil {
			sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[31m[ERROR] Unable to send input to running command.\x1b[0m\r\n"})
		}
		return
	}

	cmd := strings.TrimSpace(input)
	if cmd == "" {
		return
	}
	h.handleTerminalCommand(conn, sess, cmd)
}

func (h *Handlers) handleTerminalUnlock(conn *websocket.Conn, sess *termSession, totpCode string) {
	if totpCode == "" {
		sendTermMsg(conn, sess, termMsg{Type: "unlock_fail", Data: "No 2FA code provided."})
		return
	}

	user, err := db.GetUserByUsername(sess.username)
	if err != nil || user == nil {
		sendTermMsg(conn, sess, termMsg{Type: "unlock_fail", Data: "User lookup failed."})
		return
	}

	if !user.TOTPEnabled {
		sendTermMsg(conn, sess, termMsg{
			Type: "unlock_fail",
			Data: "Your account has no 2FA configured. Enable 2FA in Settings before using high-risk commands.",
		})
		return
	}

	if !appauth.ValidateCode(user.TOTPSecret, totpCode) {
		db.LogLoginAttempt(sess.username, sess.ip, "terminal_elevate_fail", false, "ws")
		sendTermMsg(conn, sess, termMsg{Type: "unlock_fail", Data: "Invalid 2FA code."})
		return
	}

	sess.elevate()
	db.LogLoginAttempt(sess.username, sess.ip, "terminal_elevate_ok", true, "ws")
	sendTermMsg(conn, sess, termMsg{Type: "unlocked", Data: "High-risk commands enabled for 5 minutes."})
}

func (h *Handlers) handleTerminalCommand(conn *websocket.Conn, sess *termSession, cmd string) {
	if strings.EqualFold(strings.TrimSpace(cmd), "help") {
		sendTermMsg(conn, sess, termMsg{
			Type: "output",
			Data: "Allowed without elevation: " + strings.Join(safeCommandPrefixes, ", ") + "\r\n" +
				"High-risk mode (2FA, 5 min) is required for all other commands.\r\n",
		})
		return
	}

	// Echo command
	sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[1;36m$ " + cmd + "\x1b[0m\r\n"})

	risk, matchedPat := classifyCommand(cmd)

	switch risk {
	case riskBlocked:
		sendTermMsg(conn, sess, termMsg{
			Type: "output",
			Data: "\x1b[1;31m[BLOCKED] This command is permanently disabled: " + matchedPat + "\x1b[0m\r\n",
		})
		db.LogLoginAttempt(sess.username, sess.ip, "terminal_blocked: "+cmd, false, "ws")
		return

	case riskHigh:
		if !sess.isElevated() {
			sendTermMsg(conn, sess, termMsg{Type: "need_unlock", Cmd: cmd, Data: matchedPat})
			return
		}
		db.LogLoginAttempt(sess.username, sess.ip, "terminal_highrisk: "+cmd, true, "ws")

	default:
		db.LogLoginAttempt(sess.username, sess.ip, "terminal_exec: "+cmd, true, "ws")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	execCmd := buildExecCommand(ctx, cmd)

	stdin, err := execCmd.StdinPipe()
	if err != nil {
		cancel()
		sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[31m[ERROR] Failed to open command stdin.\x1b[0m\r\n"})
		return
	}
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		cancel()
		sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[31m[ERROR] Failed to open command stdout.\x1b[0m\r\n"})
		return
	}
	stderr, err := execCmd.StderrPipe()
	if err != nil {
		_ = stdin.Close()
		cancel()
		sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[31m[ERROR] Failed to open command stderr.\x1b[0m\r\n"})
		return
	}

	if !sess.startActiveCommand(stdin, cancel) {
		_ = stdin.Close()
		cancel()
		sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[33m[INFO] Another command is already running.\x1b[0m\r\n"})
		return
	}

	if err := execCmd.Start(); err != nil {
		sess.clearActiveCommand()
		sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[31m[ERROR] Failed to start command: " + err.Error() + "\x1b[0m\r\n"})
		return
	}

	go func() {
		defer sess.clearActiveCommand()

		streamDone := make(chan struct{}, 2)
		stream := func(r io.Reader) {
			defer func() { streamDone <- struct{}{} }()
			buf := make([]byte, 2048)
			for {
				n, readErr := r.Read(buf)
				if n > 0 {
					chunk := strings.ReplaceAll(string(buf[:n]), "\n", "\r\n")
					sendTermMsg(conn, sess, termMsg{Type: "output", Data: chunk})
				}
				if readErr != nil {
					return
				}
			}
		}

		go stream(stdout)
		go stream(stderr)

		waitErr := execCmd.Wait()
		<-streamDone
		<-streamDone

		if ctx.Err() == context.DeadlineExceeded {
			sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[31m[ERROR] Command timed out after 10 minutes.\x1b[0m\r\n"})
			return
		}

		if waitErr != nil {
			sendTermMsg(conn, sess, termMsg{Type: "output", Data: "\x1b[31m[ERROR] Command exited: " + waitErr.Error() + "\x1b[0m\r\n"})
		}
	}()
}

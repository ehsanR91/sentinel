package api

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"

	appauth "github.com/ehsanR91/sentinelcore/internal/auth"
	"github.com/ehsanR91/sentinelcore/internal/config"
	"github.com/ehsanR91/sentinelcore/internal/db"
	dockerclient "github.com/ehsanR91/sentinelcore/internal/docker"
	"github.com/ehsanR91/sentinelcore/internal/monitoring"
	"github.com/ehsanR91/sentinelcore/internal/notify"
	appws "github.com/ehsanR91/sentinelcore/internal/ws"
)

var pinRegex = regexp.MustCompile(`^\d{6}$`)

// pinFailures tracks recent failed PIN attempts per user ID for rate limiting.
var (
	pinFailures   = map[int64][]int64{}
	pinFailuresMu sync.Mutex
)

const pinMaxAttempts = 5
const pinWindowSec = 120

// Handlers holds shared dependencies for all HTTP handlers.
type Handlers struct {
	cfg       *config.Config
	collector *monitoring.Collector
	hub       *appws.Hub
	mailer    *notify.Mailer
	gs        *grantStore
	proxyMu   sync.Mutex
	proxies   map[string]net.Listener
	logMu     sync.Mutex
	logHubs   map[string]*containerLogHub
}

func NewHandlers(cfg *config.Config, collector *monitoring.Collector, hub *appws.Hub, mailer *notify.Mailer) *Handlers {
	return &Handlers{
		cfg:       cfg,
		collector: collector,
		hub:       hub,
		mailer:    mailer,
		proxies:   map[string]net.Listener{},
		logHubs:   map[string]*containerLogHub{},
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func clientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.Split(fwd, ",")[0]
	}
	// strip port
	addr := r.RemoteAddr
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
}

func (h *Handlers) issueJWT(userID int64, username, role string, expiry time.Duration) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  username,
		"role": role,
		"uid":  userID,
		"exp":  time.Now().Add(expiry).Unix(),
		"iat":  time.Now().Unix(),
	})
	return tok.SignedString([]byte(h.cfg.JWTSecret))
}

func (h *Handlers) issuePendingJWT(username, role string) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  username,
		"role": role,
		"step": "2fa_pending",
		"exp":  time.Now().Add(5 * time.Minute).Unix(),
		"iat":  time.Now().Unix(),
	})
	return tok.SignedString([]byte(h.cfg.JWTSecret))
}

func parsePendingToken(tokenStr, secret string) (username, role string, err error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !tok.Valid {
		return "", "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", errors.New("bad claims")
	}
	if claims["step"] != "2fa_pending" {
		return "", "", errors.New("not a pending token")
	}
	return claims["sub"].(string), claims["role"].(string), nil
}

// ─── Auth ─────────────────────────────────────────────────────────────────────

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	ip := clientIP(r)
	ua := r.Header.Get("User-Agent")

	// Brute-force guard
	failed, _ := db.RecentFailedCount(ip, 10)
	if failed >= h.cfg.BruteForceThreshold {
		db.LogLoginAttempt(req.Username, ip, "rate_limited", false, ua)
		h.mailer.AlertBruteForce(ip)
		writeError(w, http.StatusTooManyRequests, "too many attempts — try again later")
		return
	}

	user, err := db.GetUserByUsername(req.Username)
	if err != nil || !appauth.VerifyPassword(user.PasswordHash, req.Password) {
		db.LogLoginAttempt(req.Username, ip, "bad_credentials", false, ua)
		h.mailer.AlertFailedLogin(req.Username, ip)
		writeError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	if user.TOTPEnabled {
		pending, err := h.issuePendingJWT(user.Username, user.Role)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "token error")
			return
		}
		db.LogLoginAttempt(req.Username, ip, "2fa_required", false, ua)
		writeJSON(w, http.StatusOK, map[string]any{
			"requires_2fa":  true,
			"pending_token": pending,
		})
		return
	}

	signed, csrf, err := h.issueSession(user.ID, user.Username, user.Role)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}
	h.setAuthCookies(w, r, signed, csrf)
	db.LogLoginAttempt(req.Username, ip, "", true, ua)
	writeJSON(w, http.StatusOK, map[string]any{
		"username": user.Username,
		"role":     user.Role,
	})
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	clearCookie := func(name string) {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: name == "sc_auth",
			SameSite: http.SameSiteStrictMode,
			Secure:   true,
			MaxAge:   -1,
		})
	}
	clearCookie("sc_auth")
	clearCookie("sc_csrf")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// issueSession issues a signed JWT and a random CSRF token.
func (h *Handlers) issueSession(userID int64, username, role string) (string, string, error) {
	signed, err := h.issueJWT(userID, username, role, 24*time.Hour)
	if err != nil {
		return "", "", err
	}
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		return "", "", err
	}
	csrf := base64.RawStdEncoding.EncodeToString(csrfBytes)
	return signed, csrf, nil
}

// setAuthCookies sets HttpOnly auth cookie and CSRF cookie.
func (h *Handlers) setAuthCookies(w http.ResponseWriter, r *http.Request, token, csrf string) {
	exp := time.Now().Add(24 * time.Hour)
	// Secure flag should only be true for HTTPS, false for HTTP
	// Check TLS or X-Forwarded-Proto header for HTTPS detection
	secure := r.TLS != nil
	if proto := r.Header.Get("X-Forwarded-Proto"); proto == "https" {
		secure = true
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "sc_auth",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  exp,
		MaxAge:   24 * 3600,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "sc_csrf",
		Value:    csrf,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
		Secure:   secure,
		Expires:  exp,
		MaxAge:   24 * 3600,
	})
}

// ─── 2FA ──────────────────────────────────────────────────────────────────────

type verify2FARequest struct {
	PendingToken string `json:"pending_token"`
	Code         string `json:"code"`
}

func (h *Handlers) Verify2FA(w http.ResponseWriter, r *http.Request) {
	var req verify2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	username, role, err := parsePendingToken(req.PendingToken, h.cfg.JWTSecret)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid or expired token")
		return
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not found")
		return
	}

	ip := clientIP(r)
	ua := r.Header.Get("User-Agent")

	if !appauth.ValidateCode(user.TOTPSecret, req.Code) {
		db.LogLoginAttempt(username, ip, "bad_totp", false, ua)
		h.mailer.Alert2FAFailure(username, ip)
		writeError(w, http.StatusUnauthorized, "invalid 2FA code")
		return
	}

	signed, err := h.issueJWT(user.ID, username, role, 24*time.Hour)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token error")
		return
	}
	csrfBytes := make([]byte, 32)
	if _, err := rand.Read(csrfBytes); err != nil {
		writeError(w, http.StatusInternalServerError, "csrf error")
		return
	}
	csrf := base64.RawStdEncoding.EncodeToString(csrfBytes)
	h.setAuthCookies(w, r, signed, csrf)
	db.LogLoginAttempt(username, ip, "", true, ua)
	writeJSON(w, http.StatusOK, map[string]any{
		"username": username,
		"role":     role,
	})
}

func (h *Handlers) Setup2FA(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)

	secret, otpauthURL, err := appauth.GenerateSecret(username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate secret")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"secret":      secret,
		"otpauth_url": otpauthURL,
	})
}

type enable2FARequest struct {
	Secret string `json:"secret"`
	Code   string `json:"code"`
}

func (h *Handlers) Enable2FA(w http.ResponseWriter, r *http.Request) {
	var req enable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)

	if !appauth.ValidateCode(req.Secret, req.Code) {
		writeError(w, http.StatusBadRequest, "invalid verification code")
		return
	}

	user, err := db.GetUserByUsername(username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user error")
		return
	}

	if err := db.UpdateTOTP(user.ID, req.Secret, true); err != nil {
		writeError(w, http.StatusInternalServerError, "could not enable 2FA")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

type disable2FARequest struct {
	Code string `json:"code"`
}

func (h *Handlers) Disable2FA(w http.ResponseWriter, r *http.Request) {
	var req disable2FARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)

	user, err := db.GetUserByUsername(username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user error")
		return
	}

	if !appauth.ValidateCode(user.TOTPSecret, req.Code) {
		writeError(w, http.StatusBadRequest, "invalid 2FA code")
		return
	}

	if err := db.UpdateTOTP(user.ID, "", false); err != nil {
		writeError(w, http.StatusInternalServerError, "could not disable 2FA")
		return
	}
	_ = h.recordAuditEvent(r, "auth.2fa.disabled", user.Username, "totp disabled for current account", true)
	writeJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

// ─── Me ───────────────────────────────────────────────────────────────────────

func (h *Handlers) GetMe(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)
	user, err := db.GetUserByUsername(username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "user error")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"role":         user.Role,
		"email":        user.Email,
		"totp_enabled": user.TOTPEnabled,
		"client_ip":    clientIP(r),
	})
}

// ─── Settings ─────────────────────────────────────────────────────────────────

var settingKeys = []string{
	"secret_path", "gate_expiry_days",
	"smtp_host", "smtp_port", "smtp_user", "smtp_pass", "alert_email",
	"brute_force_threshold", "email_alerts_enabled",
	"recaptcha_enabled", "recaptcha_site_key", "recaptcha_secret_key",
	"ip_lookup_provider", "ipify_api_key",
}

func (h *Handlers) GetSettings(w http.ResponseWriter, r *http.Request) {
	out := map[string]string{}
	for _, k := range settingKeys {
		switch k {
		case "smtp_pass", "recaptcha_secret_key", "ipify_api_key":
			out[k] = ""
			continue
		default:
			out[k] = db.GetSetting(k, "")
		}
	}
	if db.GetSecretSetting("smtp_pass", "") != "" {
		out["smtp_pass_configured"] = "1"
	} else {
		out["smtp_pass_configured"] = "0"
	}
	if db.GetSecretSetting("recaptcha_secret_key", "") != "" {
		out["recaptcha_secret_key_configured"] = "1"
	} else {
		out["recaptcha_secret_key_configured"] = "0"
	}
	if db.GetSecretSetting("ipify_api_key", "") != "" {
		out["ipify_api_key_configured"] = "1"
	} else {
		out["ipify_api_key_configured"] = "0"
	}
	if out["recaptcha_enabled"] == "" {
		out["recaptcha_enabled"] = "false"
	}
	if out["ip_lookup_provider"] == "" {
		out["ip_lookup_provider"] = "none"
	}
	// Fill env-based defaults for display
	if out["secret_path"] == "" {
		out["secret_path"] = h.cfg.SecretPath
	}
	if out["brute_force_threshold"] == "" {
		out["brute_force_threshold"] = strconv.Itoa(h.cfg.BruteForceThreshold)
	}
	out["secrets_key_path"] = h.cfg.SecretsKeyPath
	out["last_master_key_rotation"] = db.GetSetting("last_master_key_rotation", "")
	writeJSON(w, http.StatusOK, out)
}

func (h *Handlers) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	allowed := map[string]bool{}
	for _, k := range settingKeys {
		allowed[k] = true
	}
	for k, raw := range body {
		if !allowed[k] {
			continue
		}
		v := ""
		switch typed := raw.(type) {
		case string:
			v = typed
		case bool:
			v = strconv.FormatBool(typed)
		case float64:
			v = strconv.FormatFloat(typed, 'f', -1, 64)
		default:
			v = fmt.Sprint(typed)
		}

		if k == "smtp_pass" {
			if strings.TrimSpace(v) == "" {
				continue
			}
			if err := db.SetSecretSetting(k, v); err != nil {
				writeError(w, http.StatusInternalServerError, "could not store smtp password")
				return
			}
			h.mailer.Password = v
			continue
		}

		if k == "recaptcha_secret_key" {
			if strings.TrimSpace(v) == "" {
				continue
			}
			if err := db.SetSecretSetting(k, strings.TrimSpace(v)); err != nil {
				writeError(w, http.StatusInternalServerError, "could not store recaptcha secret")
				return
			}
			continue
		}

		if k == "ipify_api_key" {
			if strings.TrimSpace(v) == "" {
				continue
			}
			if err := db.SetSecretSetting(k, strings.TrimSpace(v)); err != nil {
				writeError(w, http.StatusInternalServerError, "could not store ipify api key")
				return
			}
			continue
		}
		if err := db.SetSetting(k, v); err != nil {
			writeError(w, http.StatusInternalServerError, "could not update settings")
			return
		}
		// Sync hot config values immediately
		switch k {
		case "secret_path":
			h.cfg.SecretPath = v
		case "brute_force_threshold":
			if n, err := strconv.Atoi(v); err == nil {
				h.cfg.BruteForceThreshold = n
			}
		case "alert_email":
			h.mailer.AlertEmail = v
		case "smtp_host":
			h.mailer.Host = v
		case "smtp_port":
			h.mailer.Port = v
		case "smtp_user":
			h.mailer.User = v
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ─── System Metrics ───────────────────────────────────────────────────────────

func (h *Handlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.collector.Latest())
}

func (h *Handlers) GetProcesses(w http.ResponseWriter, r *http.Request) {
	n := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil && v > 0 {
			n = v
		}
	}
	writeJSON(w, http.StatusOK, monitoring.TopProcesses(n))
}

func (h *Handlers) GetSuspiciousProcesses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, monitoring.DetectSuspiciousProcesses())
}

func (h *Handlers) GetServices(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, buildManagedServices(h.cfg.WatchedServices))
}

func (h *Handlers) GetDiskUsageTree(w http.ResponseWriter, r *http.Request) {
	target := strings.TrimSpace(r.URL.Query().Get("path"))
	if target == "" {
		target = "/"
	}
	target = filepath.Clean(target)
	if !filepath.IsAbs(target) {
		writeError(w, http.StatusBadRequest, "path must be absolute")
		return
	}

	depth := 2
	if q := r.URL.Query().Get("depth"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			depth = v
		}
	}
	if depth < 1 {
		depth = 1
	}
	if depth > 4 {
		depth = 4
	}

	limit := 25
	if q := r.URL.Query().Get("limit"); q != "" {
		if v, err := strconv.Atoi(q); err == nil {
			limit = v
		}
	}
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}

	// Use sudo to handle permission denied errors for directories like /var/lib/docker
	out, err := runPrivileged(context.Background(), "du", "-x", "-B1", fmt.Sprintf("--max-depth=%d", depth), target)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to inspect disk usage: "+out)
		return
	}

	type item struct {
		Path   string `json:"path"`
		Name   string `json:"name"`
		Depth  int    `json:"depth"`
		Size   uint64 `json:"size"`
		SizeHR string `json:"size_human"`
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	items := make([]item, 0, len(lines))
	var total uint64
	for _, line := range lines {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		sz, err := strconv.ParseUint(strings.TrimSpace(parts[0]), 10, 64)
		if err != nil {
			continue
		}
		p := strings.TrimSpace(parts[1])
		if p == target {
			total = sz
			continue
		}
		rel := strings.Trim(strings.TrimPrefix(p, target), "/")
		d := 1
		if rel != "" {
			d = strings.Count(rel, "/") + 1
		}
		items = append(items, item{
			Path:   p,
			Name:   filepath.Base(p),
			Depth:  d,
			Size:   sz,
			SizeHR: humanBytes(sz),
		})
	}

	sort.Slice(items, func(i, j int) bool { return items[i].Size > items[j].Size })
	if len(items) > limit {
		items = items[:limit]
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"path":        target,
		"total_size":  total,
		"total_human": humanBytes(total),
		"depth":       depth,
		"items":       items,
	})
}

// ─── Docker ───────────────────────────────────────────────────────────────────

func (h *Handlers) GetDockerInfo(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, dockerclient.Info())
}

func (h *Handlers) GetDockerContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := dockerclient.ListContainers()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "docker unavailable")
		return
	}
	writeJSON(w, http.StatusOK, containers)
}

func (h *Handlers) GetContainerStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stats, err := dockerclient.ContainerStatsOne(id)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "container stats unavailable")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handlers) ContainerStart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := dockerclient.StartContainer(id); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (h *Handlers) ContainerStop(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := dockerclient.StopContainer(id); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (h *Handlers) ContainerRestart(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := dockerclient.RestartContainer(id); err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func (h *Handlers) DockerPrune(w http.ResponseWriter, r *http.Request) {
	kind := strings.TrimSpace(chi.URLParam(r, "kind"))
	if kind == "" {
		kind = "all"
	}
	res, err := dockerclient.PruneUnused(kind)
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

var dockerLogUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return strings.HasPrefix(origin, "http://"+r.Host) || strings.HasPrefix(origin, "https://"+r.Host)
	},
}

type containerLogMessage struct {
	Type      string `json:"type"`
	Stream    string `json:"stream,omitempty"`
	Text      string `json:"text,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
	Error     string `json:"error,omitempty"`
	Status    string `json:"status,omitempty"`
}

type containerLogHub struct {
	id            string
	mu            sync.Mutex
	clients       map[*websocket.Conn]struct{}
	stop          chan struct{}
	active        bool
	lastTimestamp string
}

func (h *Handlers) getContainerLogHub(id string) *containerLogHub {
	h.logMu.Lock()
	defer h.logMu.Unlock()
	hub, ok := h.logHubs[id]
	if !ok {
		hub = &containerLogHub{
			id:      id,
			clients: map[*websocket.Conn]struct{}{},
			stop:    make(chan struct{}),
		}
		h.logHubs[id] = hub
	}
	return hub
}

func (h *Handlers) removeContainerLogHub(id string) {
	h.logMu.Lock()
	defer h.logMu.Unlock()
	delete(h.logHubs, id)
}

func (hub *containerLogHub) addClient(conn *websocket.Conn) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	hub.clients[conn] = struct{}{}
}

func (hub *containerLogHub) removeClient(conn *websocket.Conn) {
	hub.mu.Lock()
	defer hub.mu.Unlock()
	delete(hub.clients, conn)
	if len(hub.clients) == 0 {
		select {
		case <-hub.stop:
			return
		default:
			close(hub.stop)
		}
	}
}

func (hub *containerLogHub) broadcast(msg containerLogMessage) {
	hub.mu.Lock()
	for conn := range hub.clients {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.WriteJSON(msg); err != nil {
			_ = conn.Close()
			delete(hub.clients, conn)
		}
	}
	hub.mu.Unlock()
}

func (h *Handlers) GetContainerLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing container id")
		return
	}
	query := r.URL.Query()
	stdout := query.Get("stdout") != "0"
	stderr := query.Get("stderr") != "0"
	if !stdout && !stderr {
		stdout = true
		stderr = true
	}
	tail := 100
	if v := query.Get("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			tail = n
		}
	}
	opts := dockerclient.ContainerLogsOptions{
		Stdout:     stdout,
		Stderr:     stderr,
		Since:      query.Get("since"),
		Until:      query.Get("until"),
		Tail:       tail,
		Timestamps: true,
		Follow:     false,
	}
	lines, err := dockerclient.ContainerLogs(id, opts)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}

func (h *Handlers) ContainerLogsWS(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing container id")
		return
	}
	conn, err := dockerLogUpgrader.Upgrade(w, r, nil)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	hub := h.getContainerLogHub(id)
	hub.addClient(conn)
	hub.broadcast(containerLogMessage{Type: "status", Status: "attached"})
	go h.containerLogsWSReadPump(conn, hub)
	if !hub.active {
		go h.runContainerLogsHub(hub)
	}
}

func (h *Handlers) containerLogsWSReadPump(conn *websocket.Conn, hub *containerLogHub) {
	defer func() {
		hub.removeClient(conn)
		_ = conn.Close()
		if len(hub.clients) == 0 {
			h.removeContainerLogHub(hub.id)
		}
	}()
	conn.SetReadLimit(512)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (h *Handlers) runContainerLogsHub(hub *containerLogHub) {
	hub.mu.Lock()
	if hub.active {
		hub.mu.Unlock()
		return
	}
	hub.active = true
	hub.stop = make(chan struct{})
	hub.mu.Unlock()
	defer h.removeContainerLogHub(hub.id)
	backoff := 1 * time.Second
	for {
		select {
		case <-hub.stop:
			return
		default:
		}
		hub.broadcast(containerLogMessage{Type: "status", Status: "connecting"})
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			select {
			case <-hub.stop:
				cancel()
			case <-ctx.Done():
			}
		}()
		opts := dockerclient.ContainerLogsOptions{
			Stdout:     true,
			Stderr:     true,
			Since:      hub.lastTimestamp,
			Tail:       100,
			Timestamps: true,
			Follow:     true,
		}
		if hub.lastTimestamp == "" {
			opts.Tail = 100
		}
		reader, err := dockerclient.ContainerLogsStream(ctx, hub.id, opts)
		if err != nil {
			hub.broadcast(containerLogMessage{Type: "error", Error: err.Error()})
			cancel()
			hub.broadcast(containerLogMessage{Type: "status", Status: "reconnecting"})
			select {
			case <-hub.stop:
				return
			case <-time.After(backoff):
			}
			backoff *= 2
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			continue
		}
		hub.broadcast(containerLogMessage{Type: "status", Status: "connected"})
		err = h.streamContainerLogs(ctx, hub, reader)
		reader.Close()
		cancel()
		if err != nil {
			hub.broadcast(containerLogMessage{Type: "error", Error: err.Error()})
		}
		select {
		case <-hub.stop:
			return
		case <-time.After(backoff):
		}
		backoff *= 2
		if backoff > 30*time.Second {
			backoff = 30 * time.Second
		}
		hub.broadcast(containerLogMessage{Type: "status", Status: "reconnecting"})
	}
}

func (h *Handlers) streamContainerLogs(ctx context.Context, hub *containerLogHub, reader io.ReadCloser) error {
	in := make(chan dockerclient.DockerLogLine, 128)
	errCh := make(chan error, 1)
	go func() {
		errCh <- dockerclient.StreamDockerLogLines(ctx, reader, in)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return err
		case line, ok := <-in:
			if !ok {
				return nil
			}
			if line.Timestamp != "" {
				hub.lastTimestamp = line.Timestamp
			}
			hub.broadcast(containerLogMessage{Type: "line", Stream: line.Stream, Timestamp: line.Timestamp, Text: line.Text})
		}
	}
}

// ─── Lock Screen PIN ──────────────────────────────────────────────────────────

func (h *Handlers) GetLockSettings(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	userIDFloat, ok := claims["uid"].(float64)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid user")
		return
	}
	userID := int64(userIDFloat)

	var enabled int
	var pinHash string
	err := db.DB().QueryRow(`
		SELECT enabled, pin_hash
		FROM user_lock_settings
		WHERE user_id = ?
	`, userID).Scan(&enabled, &pinHash)

	if err == sql.ErrNoRows {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": false,
			"pinSet":  false,
		})
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"enabled": enabled == 1,
		"pinSet":  pinHash != "",
	})
}

func (h *Handlers) SaveLockPin(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	userIDFloat, ok := claims["uid"].(float64)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid user")
		return
	}
	userID := int64(userIDFloat)

	var req struct {
		PIN     string `json:"pin"`
		Enabled bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	if len(req.PIN) != 6 || !pinRegex.MatchString(req.PIN) {
		writeError(w, http.StatusBadRequest, "PIN must be exactly 6 digits")
		return
	}

	hashBytes, err := bcrypt.GenerateFromPassword([]byte(req.PIN), bcrypt.DefaultCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not hash PIN")
		return
	}
	hash := string(hashBytes)
	enabled := 0
	if req.Enabled {
		enabled = 1
	}

	_, err = db.DB().Exec(`
		INSERT INTO user_lock_settings (user_id, enabled, pin_hash)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			enabled = excluded.enabled,
			pin_hash = excluded.pin_hash
	`, userID, enabled, hash)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.recordAuditEvent(r, "lock_pin.saved", strconv.FormatInt(userID, 10), fmt.Sprintf("enabled=%t", req.Enabled), true)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) ClearLockPin(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	userIDFloat, ok := claims["uid"].(float64)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid user")
		return
	}
	userID := int64(userIDFloat)

	_, err := db.DB().Exec(`
		DELETE FROM user_lock_settings WHERE user_id = ?
	`, userID)

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	_ = h.recordAuditEvent(r, "lock_pin.cleared", strconv.FormatInt(userID, 10), "lock pin removed", true)

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) VerifyLockPin(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r)
	userIDFloat, ok := claims["uid"].(float64)
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid user")
		return
	}
	userID := int64(userIDFloat)

	var req struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}

	// Rate limit: max 5 failures per 2-minute window per user.
	now := time.Now().Unix()
	pinFailuresMu.Lock()
	recent := pinFailures[userID]
	cutoff := now - pinWindowSec
	filtered := recent[:0]
	for _, ts := range recent {
		if ts > cutoff {
			filtered = append(filtered, ts)
		}
	}
	pinFailures[userID] = filtered
	if len(filtered) >= pinMaxAttempts {
		pinFailuresMu.Unlock()
		writeError(w, http.StatusTooManyRequests, "too many PIN attempts — try again later")
		return
	}
	pinFailuresMu.Unlock()

	var storedHash string
	err := db.DB().QueryRow(`
		SELECT pin_hash FROM user_lock_settings WHERE user_id = ?
	`, userID).Scan(&storedHash)

	if err == sql.ErrNoRows {
		writeError(w, http.StatusNotFound, "no PIN set")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.PIN)) == nil {
		// Success: clear failure counter.
		pinFailuresMu.Lock()
		delete(pinFailures, userID)
		pinFailuresMu.Unlock()
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	} else {
		// Record failure.
		pinFailuresMu.Lock()
		pinFailures[userID] = append(pinFailures[userID], now)
		pinFailuresMu.Unlock()
		writeError(w, http.StatusUnauthorized, "incorrect PIN")
	}
}

// ─── WebSocket ────────────────────────────────────────────────────────────────

func (h *Handlers) WSConnect(w http.ResponseWriter, r *http.Request) {
	appws.ServeWS(h.hub, w, r)
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func humanBytes(v uint64) string {
	if v >= 1<<40 {
		return fmt.Sprintf("%.1f TB", float64(v)/(1<<40))
	}
	if v >= 1<<30 {
		return fmt.Sprintf("%.1f GB", float64(v)/(1<<30))
	}
	if v >= 1<<20 {
		return fmt.Sprintf("%.1f MB", float64(v)/(1<<20))
	}
	if v >= 1<<10 {
		return fmt.Sprintf("%.0f KB", float64(v)/(1<<10))
	}
	return fmt.Sprintf("%d B", v)
}

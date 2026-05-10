package api

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ehsanR91/sentinelcore/internal/db"
)

type tlsSettingsResponse struct {
	Enabled    bool   `json:"enabled"`
	Mode       string `json:"mode"`
	CertPath   string `json:"cert_path"`
	KeyPath    string `json:"key_path"`
	ServerName string `json:"server_name"`
	AutoRenew  bool   `json:"auto_renew"`
	UpdatedAt  int64  `json:"updated_at"`
}

type ipAllowlistResponse struct {
	Enabled   bool     `json:"enabled"`
	Entries   []string `json:"entries"`
	UpdatedAt int64    `json:"updated_at"`
}

type notificationRoute struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Channel  string `json:"channel"`
	Target   string `json:"target"`
	Enabled  bool   `json:"enabled"`
}

type notificationRoutingResponse struct {
	Routes    []notificationRoute `json:"routes"`
	UpdatedAt int64               `json:"updated_at"`
}

var allowedSeverities = map[string]bool{
	"info": true, "warn": true, "error": true, "critical": true,
}

var allowedChannels = map[string]bool{
	"email": true, "webhook": true, "slack": true, "telegram": true,
}

func (h *Handlers) GetTLSSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, loadTLSSettings())
}

func (h *Handlers) UpdateTLSSettings(w http.ResponseWriter, r *http.Request) {
	var req tlsSettingsResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	mode := strings.TrimSpace(req.Mode)
	if mode == "" {
		mode = "terminated"
	}
	if mode != "terminated" && mode != "passthrough" {
		writeError(w, http.StatusBadRequest, "invalid tls mode")
		return
	}
	if req.Enabled && (strings.TrimSpace(req.CertPath) == "" || strings.TrimSpace(req.KeyPath) == "") {
		writeError(w, http.StatusBadRequest, "certificate and key paths are required when TLS is enabled")
		return
	}
	updatedAt := time.Now().Unix()
	settings := map[string]string{
		"tls_enabled":     boolString(req.Enabled),
		"tls_mode":        mode,
		"tls_cert_path":   strings.TrimSpace(req.CertPath),
		"tls_key_path":    strings.TrimSpace(req.KeyPath),
		"tls_server_name": strings.TrimSpace(req.ServerName),
		"tls_auto_renew":  boolString(req.AutoRenew),
		"tls_updated_at":  fmt.Sprintf("%d", updatedAt),
	}
	for key, value := range settings {
		if err := db.SetSetting(key, value); err != nil {
			writeError(w, http.StatusInternalServerError, "could not persist TLS settings")
			return
		}
	}
	_ = h.recordAuditEvent(r, "settings.tls.update", "tls", fmt.Sprintf("mode=%s enabled=%t", mode, req.Enabled), true)
	writeJSON(w, http.StatusOK, loadTLSSettings())
}

func (h *Handlers) GetIPAllowlist(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, loadIPAllowlist())
}

func (h *Handlers) UpdateIPAllowlist(w http.ResponseWriter, r *http.Request) {
	var req ipAllowlistResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	entries := normalizeAllowlistEntries(req.Entries)
	for _, entry := range entries {
		if !isValidAllowlistEntry(entry) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid allowlist entry: %s", entry))
			return
		}
	}
	encoded, err := json.Marshal(entries)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encoding error")
		return
	}
	updatedAt := time.Now().Unix()
	if err := db.SetSetting("ip_allowlist_enabled", boolString(req.Enabled)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist allowlist")
		return
	}
	if err := db.SetSetting("ip_allowlist_entries", string(encoded)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist allowlist")
		return
	}
	if err := db.SetSetting("ip_allowlist_updated_at", fmt.Sprintf("%d", updatedAt)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist allowlist")
		return
	}
	_ = h.recordAuditEvent(r, "settings.allowlist.update", "ip_allowlist", fmt.Sprintf("entries=%d enabled=%t", len(entries), req.Enabled), true)
	writeJSON(w, http.StatusOK, loadIPAllowlist())
}

func (h *Handlers) GetNotificationRouting(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, loadNotificationRouting())
}

func (h *Handlers) UpdateNotificationRouting(w http.ResponseWriter, r *http.Request) {
	var req notificationRoutingResponse
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	for index := range req.Routes {
		route := &req.Routes[index]
		route.ID = strings.TrimSpace(route.ID)
		route.Name = strings.TrimSpace(route.Name)
		route.Severity = strings.TrimSpace(strings.ToLower(route.Severity))
		route.Channel = strings.TrimSpace(strings.ToLower(route.Channel))
		route.Target = strings.TrimSpace(route.Target)
		if route.ID == "" {
			route.ID = fmt.Sprintf("route-%d", index+1)
		}
		if route.Name == "" {
			route.Name = fmt.Sprintf("Route %d", index+1)
		}
		if !allowedSeverities[route.Severity] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid severity for route %s", route.Name))
			return
		}
		if !allowedChannels[route.Channel] {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid channel for route %s", route.Name))
			return
		}
		if route.Target == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("target is required for route %s", route.Name))
			return
		}
	}
	sort.Slice(req.Routes, func(i, j int) bool { return req.Routes[i].ID < req.Routes[j].ID })
	encoded, err := json.Marshal(req.Routes)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "encoding error")
		return
	}
	updatedAt := time.Now().Unix()
	if err := db.SetSetting("notification_routes", string(encoded)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist notification routing")
		return
	}
	if err := db.SetSetting("notification_routes_updated_at", fmt.Sprintf("%d", updatedAt)); err != nil {
		writeError(w, http.StatusInternalServerError, "could not persist notification routing")
		return
	}
	_ = h.recordAuditEvent(r, "settings.notification_routing.update", "notification_routes", fmt.Sprintf("routes=%d", len(req.Routes)), true)
	writeJSON(w, http.StatusOK, loadNotificationRouting())
}

func (h *Handlers) RotateMasterKey(w http.ResponseWriter, r *http.Request) {
	if strings.TrimSpace(h.cfg.SecretsKeyPath) == "" {
		writeError(w, http.StatusBadRequest, "secrets key path is not configured")
		return
	}
	if err := db.RotateSecretsKey(h.cfg.SecretsKeyPath); err != nil {
		_ = h.recordAuditEvent(r, "settings.master_key.rotate", h.cfg.SecretsKeyPath, err.Error(), false)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rotatedAt := db.GetSetting("last_master_key_rotation", "")
	invalidateSettingsResponseCache()
	_ = h.recordAuditEvent(r, "settings.master_key.rotate", h.cfg.SecretsKeyPath, fmt.Sprintf("rotated_at=%s", rotatedAt), true)
	writeJSON(w, http.StatusOK, map[string]string{
		"status":                   "ok",
		"last_master_key_rotation": rotatedAt,
	})
}

func (h *Handlers) recordAuditEvent(r *http.Request, action, target, details string, success bool) error {
	claims := claimsFromCtx(r)
	username, _ := claims["sub"].(string)
	ip := clientIP(r)
	return db.InsertAuditEvent(username, ip, action, target, details, success)
}

func loadTLSSettings() tlsSettingsResponse {
	return tlsSettingsResponse{
		Enabled:    db.GetSetting("tls_enabled", "false") == "true",
		Mode:       defaultString(db.GetSetting("tls_mode", "terminated"), "terminated"),
		CertPath:   db.GetSetting("tls_cert_path", ""),
		KeyPath:    db.GetSetting("tls_key_path", ""),
		ServerName: db.GetSetting("tls_server_name", ""),
		AutoRenew:  db.GetSetting("tls_auto_renew", "false") == "true",
		UpdatedAt:  parseUnix(db.GetSetting("tls_updated_at", "0")),
	}
}

func loadIPAllowlist() ipAllowlistResponse {
	entries := []string{}
	if raw := strings.TrimSpace(db.GetSetting("ip_allowlist_entries", "[]")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &entries)
	}
	return ipAllowlistResponse{
		Enabled:   db.GetSetting("ip_allowlist_enabled", "false") == "true",
		Entries:   normalizeAllowlistEntries(entries),
		UpdatedAt: parseUnix(db.GetSetting("ip_allowlist_updated_at", "0")),
	}
}

func loadNotificationRouting() notificationRoutingResponse {
	routes := []notificationRoute{}
	if raw := strings.TrimSpace(db.GetSetting("notification_routes", "[]")); raw != "" {
		_ = json.Unmarshal([]byte(raw), &routes)
	}
	if len(routes) == 0 {
		routes = []notificationRoute{{
			ID:       "email-critical",
			Name:     "Critical Email",
			Severity: "critical",
			Channel:  "email",
			Target:   db.GetSetting("alert_email", ""),
			Enabled:  strings.TrimSpace(db.GetSetting("alert_email", "")) != "",
		}}
	}
	return notificationRoutingResponse{
		Routes:    routes,
		UpdatedAt: parseUnix(db.GetSetting("notification_routes_updated_at", "0")),
	}
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func parseUnix(raw string) int64 {
	if raw == "" {
		return 0
	}
	var ts int64
	fmt.Sscanf(raw, "%d", &ts)
	return ts
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func normalizeAllowlistEntries(entries []string) []string {
	seen := map[string]struct{}{}
	normalized := make([]string, 0, len(entries))
	for _, entry := range entries {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func isValidAllowlistEntry(entry string) bool {
	if ip := net.ParseIP(entry); ip != nil {
		return true
	}
	_, _, err := net.ParseCIDR(entry)
	return err == nil
}

func ipMatchesAllowlist(ip string, entries []string) bool {
	parsedIP := net.ParseIP(strings.TrimSpace(ip))
	if parsedIP == nil {
		return false
	}
	if parsedIP.IsLoopback() {
		return true
	}
	for _, entry := range entries {
		if candidate := net.ParseIP(entry); candidate != nil && candidate.Equal(parsedIP) {
			return true
		}
		if _, network, err := net.ParseCIDR(entry); err == nil && network.Contains(parsedIP) {
			return true
		}
	}
	return false
}

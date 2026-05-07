package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// serviceVerifyCmd maps a service name to the privileged command used to test its config.
var serviceVerifyCmd = map[string][]string{
	"nginx":    {"nginx", "-t"},
	"sshd":     {"sshd", "-t"},
	"fail2ban": {"fail2ban-client", "--test"},
}

func (h *Handlers) GetServiceConfigFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := serviceCatalog[name]
	if !ok || meta.Config == "" {
		writeError(w, http.StatusBadRequest, "no config file for this service")
		return
	}
	content, err := readConfigFile(r.Context(), meta.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": meta.Config, "content": content})
}

func (h *Handlers) SaveServiceConfigFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := serviceCatalog[name]
	if !ok || meta.Config == "" {
		writeError(w, http.StatusBadRequest, "no config file for this service")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := writeConfigFile(r.Context(), meta.Config, req.Content); err != nil {
		writeError(w, http.StatusInternalServerError, "could not write config: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved", "path": meta.Config})
}

func (h *Handlers) BackupServiceConfigFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := serviceCatalog[name]
	if !ok || meta.Config == "" {
		writeError(w, http.StatusBadRequest, "no config file for this service")
		return
	}
	content, err := readConfigFile(r.Context(), meta.Config)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read config: "+err.Error())
		return
	}
	backupPath := fmt.Sprintf("%s.bak.%d", meta.Config, time.Now().Unix())
	if err := writeConfigFile(r.Context(), backupPath, content); err != nil {
		writeError(w, http.StatusInternalServerError, "could not write backup: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "backed_up", "backup_path": backupPath})
}

func (h *Handlers) VerifyServiceConfigFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if _, ok := serviceCatalog[name]; !ok {
		writeError(w, http.StatusBadRequest, "unsupported service")
		return
	}
	vcmd, ok := serviceVerifyCmd[name]
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"valid": true, "message": "no verifier available for this service"})
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	out, err := runPrivileged(ctx, vcmd[0], vcmd[1:]...)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"valid": false, "message": strings.TrimSpace(out)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true, "message": strings.TrimSpace(out)})
}

func (h *Handlers) RestoreServiceConfigFile(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	meta, ok := serviceCatalog[name]
	if !ok || meta.Config == "" {
		writeError(w, http.StatusBadRequest, "no config file for this service")
		return
	}
	dir := filepath.Dir(meta.Config)
	base := filepath.Base(meta.Config)
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not list backups: "+err.Error())
		return
	}
	var latest string
	var latestTs int64
	for _, e := range entries {
		n := e.Name()
		if !strings.HasPrefix(n, base+".bak.") {
			continue
		}
		var ts int64
		fmt.Sscanf(strings.TrimPrefix(n, base+".bak."), "%d", &ts)
		if ts > latestTs {
			latestTs = ts
			latest = filepath.Join(dir, n)
		}
	}
	if latest == "" {
		writeError(w, http.StatusNotFound, "no backup found")
		return
	}
	content, err := readConfigFile(r.Context(), latest)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not read backup: "+err.Error())
		return
	}
	if err := writeConfigFile(r.Context(), meta.Config, content); err != nil {
		writeError(w, http.StatusInternalServerError, "could not restore: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restored", "from": latest})
}

// readConfigFile reads a (potentially root-owned) config file via sudo cat.
func readConfigFile(ctx context.Context, path string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := runPrivileged(ctx, "cat", path)
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(out))
	}
	return out, nil
}

// writeConfigFile writes content to a system config file via sudo tee.
// tee reads from stdin and writes to the path, preserving permissions.
func writeConfigFile(ctx context.Context, path string, content string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var c *exec.Cmd
	if isRootUser() {
		c = exec.CommandContext(ctx, "tee", path)
	} else {
		c = exec.CommandContext(ctx, "sudo", "-n", "tee", path)
	}
	c.Stdin = strings.NewReader(content)
	// tee also writes to stdout — discard it, capture stderr via CombinedOutput.
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return nil
}

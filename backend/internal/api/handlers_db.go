package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ehsanR91/sentinelcore/internal/db"
)

func (h *Handlers) GetDBStats(w http.ResponseWriter, r *http.Request) {
	var loginCount, alertCount, manualBanCount int64
	db.DB().QueryRow("SELECT COUNT(*) FROM login_attempts").Scan(&loginCount)
	db.DB().QueryRow("SELECT COUNT(*) FROM alerts").Scan(&alertCount)
	db.DB().QueryRow("SELECT COUNT(*) FROM manual_bans").Scan(&manualBanCount)

	writeJSON(w, http.StatusOK, map[string]any{
		"login_attempts": loginCount,
		"alerts":         alertCount,
		"manual_bans":    manualBanCount,
	})
}

func (h *Handlers) ExportDB(w http.ResponseWriter, r *http.Request) {
	dbPath := h.cfg.DBPath
	w.Header().Set("Content-Disposition", `attachment; filename="sentinelcore.db"`)
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, dbPath)
}

func (h *Handlers) ImportDB(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 256<<20) // 256 MB max
	if err := r.ParseMultipartForm(256 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse upload")
		return
	}
	file, header, err := r.FormFile("db")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing db file in form")
		return
	}
	defer file.Close()

	// Write to a temp file first, then atomically replace
	tmp, err := os.CreateTemp("", "sentinelcore-import-*.db")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not create temp file")
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err = io.Copy(tmp, file); err != nil {
		tmp.Close()
		writeError(w, http.StatusInternalServerError, "upload failed")
		return
	}
	tmp.Close()

	// Close the current DB, replace file, reopen
	if err = db.CloseAndReplace(tmpName, h.cfg.DBPath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to replace database")
		return
	}

	_ = h.recordAuditEvent(r, "db.import", header.Filename, "database snapshot replaced", true)

	writeJSON(w, http.StatusOK, map[string]any{"message": "database imported successfully"})
}

type pruneRequest struct {
	Type string `json:"type"` // "login_attempts" or "alerts"
	Days int    `json:"days"` // keep last N days (0 = delete all)
}

func (h *Handlers) PruneDB(w http.ResponseWriter, r *http.Request) {
	var req pruneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Type != "login_attempts" && req.Type != "alerts" {
		writeError(w, http.StatusBadRequest, "type must be login_attempts or alerts")
		return
	}

	var cutoff int64
	if req.Days > 0 {
		cutoff = time.Now().Add(-time.Duration(req.Days) * 24 * time.Hour).Unix()
	}

	var affected int64
	var result interface{ RowsAffected() (int64, error) }
	var err error

	if req.Days <= 0 {
		result, err = db.DB().Exec("DELETE FROM " + req.Type)
	} else {
		result, err = db.DB().Exec("DELETE FROM "+req.Type+" WHERE ts < ?", cutoff)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, "prune failed")
		return
	}
	affected, _ = result.RowsAffected()
	_ = h.recordAuditEvent(r, "db.prune", req.Type, "deleted="+strconv.FormatInt(affected, 10), true)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": affected})
}

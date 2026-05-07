package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	appauth "github.com/ehsanR91/sentinelcore/internal/auth"
	"github.com/ehsanR91/sentinelcore/internal/db"
)

func (h *Handlers) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := db.ListUsers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	// Strip password hashes and TOTP secrets before returning
	type safeUser struct {
		ID          int64  `json:"id"`
		Username    string `json:"username"`
		Role        string `json:"role"`
		Email       string `json:"email"`
		TOTPEnabled bool   `json:"totp_enabled"`
		CreatedAt   int64  `json:"created_at"`
		UpdatedAt   int64  `json:"updated_at"`
	}
	out := make([]safeUser, 0, len(users))
	for _, u := range users {
		out = append(out, safeUser{
			ID:          u.ID,
			Username:    u.Username,
			Role:        u.Role,
			Email:       u.Email,
			TOTPEnabled: u.TOTPEnabled,
			CreatedAt:   u.CreatedAt,
			UpdatedAt:   u.UpdatedAt,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type createUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
	Email    string `json:"email"`
}

func (h *Handlers) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "username and password required")
		return
	}
	if len(req.Password) < 8 {
		writeError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	if req.Role == "" {
		req.Role = "viewer"
	}
	hash, err := appauth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hash error")
		return
	}
	if err := db.CreateUser(req.Username, hash, req.Role, req.Email); err != nil {
		writeError(w, http.StatusConflict, "username already exists")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "created"})
}

type updateUserRequest struct {
	Role  string `json:"role"`
	Email string `json:"email"`
}

func (h *Handlers) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if err := db.UpdateUser(id, req.Role, req.Email); err != nil {
		writeError(w, http.StatusInternalServerError, "update failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handlers) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}
	claims := claimsFromCtx(r)
	// Prevent self-deletion
	if me, _ := db.GetUserByUsername(claims["sub"].(string)); me != nil && me.ID == id {
		writeError(w, http.StatusBadRequest, "cannot delete your own account")
		return
	}
	if err := db.DeleteUser(id); err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

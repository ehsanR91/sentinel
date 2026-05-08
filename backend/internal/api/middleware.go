package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"

	"github.com/ehsanR91/sentinelcore/internal/db"
)

type contextKey string

const claimsKey contextKey = "claims"

// RequireAuth validates the JWT from Authorization header or secure auth cookie and injects claims into context.
func RequireAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				if c, err := r.Cookie("sc_auth"); err == nil && c.Value != "" {
					authHeader = "Bearer " + c.Value
				}
			}
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing token")
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			})
			if err != nil || !token.Valid {
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeError(w, http.StatusUnauthorized, "invalid claims")
				return
			}

			// Reject tokens that are only valid for the 2FA pending step.
			if claims["step"] == "2fa_pending" {
				writeError(w, http.StatusUnauthorized, "2FA verification required")
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// CSRFProtect enforces double-submit cookie rule on unsafe methods.
func CSRFProtect() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			method := strings.ToUpper(r.Method)
			if method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions {
				next.ServeHTTP(w, r)
				return
			}
			csrfCookie, err := r.Cookie("sc_csrf")
			if err != nil || csrfCookie.Value == "" {
				writeError(w, http.StatusForbidden, "csrf token missing")
				return
			}
			header := r.Header.Get("X-CSRF-Token")
			if header == "" || header != csrfCookie.Value {
				writeError(w, http.StatusForbidden, "csrf token mismatch")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole rejects requests whose JWT role claim is not in the allowed set.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(claimsKey).(jwt.MapClaims)
			if !ok {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			role, _ := claims["role"].(string)
			if !allowed[role] {
				writeError(w, http.StatusForbidden, "insufficient privileges")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// claimsFromCtx extracts JWT claims injected by RequireAuth.
func claimsFromCtx(r *http.Request) jwt.MapClaims {
	claims, _ := r.Context().Value(claimsKey).(jwt.MapClaims)
	return claims
}

// SecurityHeaders adds security response headers to every response.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// CDN allowlist: MDI icons (cdn.jsdelivr.net), remixicon (cdn.jsdelivr.net),
		// typicons (cdnjs.cloudflare.com). Both style sheets and their font files are allowed.
		const cdnStyles = "https://cdn.jsdelivr.net https://cdnjs.cloudflare.com"
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self' 'unsafe-inline' 'unsafe-eval'; "+
				"style-src 'self' 'unsafe-inline' "+cdnStyles+"; "+
				"font-src 'self' data: "+cdnStyles+"; "+
				"img-src 'self' data:; "+
				"connect-src 'self' ws: wss: https://cdn.jsdelivr.net https://cdnjs.cloudflare.com; "+
				"frame-ancestors 'none';")
		// HSTS: 1 year, include subdomains. Only meaningful over HTTPS but harmless over HTTP.
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
	})
}

func IPAllowlist() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if db.GetSetting("ip_allowlist_enabled", "false") != "true" {
				next.ServeHTTP(w, r)
				return
			}
			entries := []string{}
			_ = json.Unmarshal([]byte(db.GetSetting("ip_allowlist_entries", "[]")), &entries)
			if ipMatchesAllowlist(clientIP(r), entries) {
				next.ServeHTTP(w, r)
				return
			}
			writeError(w, http.StatusForbidden, "client IP is not in the configured allowlist")
		})
	}
}

package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ehsanR91/sentinelcore/internal/db"
)

const gateCookieName = "sc_gate"
const gateHMACData = "sentinel-gate-v1"

var (
	secretMu   sync.RWMutex
	cachedPath string
	cachedHMAC string
)

// computeGateHMAC creates a deterministic token from the secret key + path.
func computeGateHMAC(jwtSecret, secretPath string) string {
	h := hmac.New(sha256.New, []byte(jwtSecret+secretPath))
	h.Write([]byte(gateHMACData))
	return hex.EncodeToString(h.Sum(nil))
}

// gatePassthroughPaths are fetched by the browser without gate cookies (PWA
// manifest, service worker) and must be accessible without the gate.
var gatePassthroughPaths = map[string]bool{
	"/manifest.webmanifest": true,
	"/sw.js":                true,
}

// GateMiddleware blocks all requests that don't carry a valid sc_gate cookie.
// The activation route must be registered separately (see ActivateGateHandler).
func GateMiddleware(jwtSecret string, isDev bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow PWA/service-worker assets through without gate cookie.
                        if isDev {
                                next.ServeHTTP(w, r)
                                return
                        }

			if gatePassthroughPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Re-check the current secret path from DB on every request (SQLite is fast).
			currentPath := db.GetSetting("secret_path", "sentinel-core")
			expected := computeGateHMAC(jwtSecret, currentPath)

			// Check for exact path match → this IS the activation request.
			// (Also handles trailing-slash variant.)
			reqPath := r.URL.Path
			activationPath := "/" + currentPath
			if reqPath == activationPath || reqPath == activationPath+"/" {
				activateAndRedirect(w, r, jwtSecret, currentPath)
				return
			}

			cookie, err := r.Cookie(gateCookieName)
			if err != nil || cookie.Value != expected {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(accessDeniedHTML))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func activateAndRedirect(w http.ResponseWriter, r *http.Request, jwtSecret, secretPath string) {
	token := computeGateHMAC(jwtSecret, secretPath)

	// Read expiry from DB settings (days).
	expiryDays, _ := strconv.Atoi(db.GetSetting("gate_expiry_days", "0"))

	cookie := &http.Cookie{
		Name:     gateCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}
	if expiryDays > 0 {
		cookie.Expires = time.Now().Add(time.Duration(expiryDays) * 24 * time.Hour)
		cookie.MaxAge = expiryDays * 86400
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, "/", http.StatusFound)
}

const accessDeniedHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Access Denied</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{background:#000;color:#333;font-family:monospace;display:flex;align-items:center;justify-content:center;min-height:100vh}
.box{text-align:center;user-select:none}
.code{font-size:6rem;font-weight:900;color:#1a1a1a;letter-spacing:-4px}
.sep{width:60px;height:2px;background:#111;margin:1.2rem auto}
.msg{font-size:0.75rem;color:#222;letter-spacing:.2em;text-transform:uppercase}
</style>
</head>
<body>
<div class="box">
  <div class="code">403</div>
  <div class="sep"></div>
  <div class="msg">Access Denied</div>
</div>
</body>
</html>`

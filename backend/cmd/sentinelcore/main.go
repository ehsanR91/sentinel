package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/ehsanR91/sentinelcore/internal/api"
	appauth "github.com/ehsanR91/sentinelcore/internal/auth"
	"github.com/ehsanR91/sentinelcore/internal/config"
	"github.com/ehsanR91/sentinelcore/internal/db"
	"github.com/ehsanR91/sentinelcore/internal/monitoring"
	"github.com/ehsanR91/sentinelcore/internal/notify"
	appws "github.com/ehsanR91/sentinelcore/internal/ws"
)

func main() {
	// Admin maintenance subcommand: sentinelcore admin <cmd> [args]
	if len(os.Args) > 1 && os.Args[1] == "admin" {
		runAdminTool(os.Args[2:])
		return
	}

	cfg := config.Load()
	if cfg.JWTSecret == "" || cfg.JWTSecret == "change-me-in-production" || len(cfg.JWTSecret) < 32 {
		log.Fatalf("JWT_SECRET is not set or is the default placeholder. Set a random secret of at least 32 characters in .env")
	}

	// ── Database ───────────────────────────────────────────────────────────────
	if err := db.Init(cfg.DBPath, cfg.SecretsKeyPath); err != nil {
		log.Fatalf("DB init failed: %v", err)
	}

	adminPass := cfg.InitialAdminPass
	if adminPass == "" && cfg.InitialAdminPassFile != "" {
		if _, err := os.Stat(cfg.InitialAdminPassFile); err == nil {
			raw, readErr := os.ReadFile(cfg.InitialAdminPassFile)
			if readErr != nil {
				log.Printf("Initial admin password file read failed (%s): %v", cfg.InitialAdminPassFile, readErr)
			} else {
				adminPass = strings.TrimSpace(string(raw))
			}
		}
	}

	// Seed admin user on first start
	if adminPass != "" {
		hash, err := appauth.HashPassword(adminPass)
		if err != nil {
			log.Fatalf("Password hash failed: %v", err)
		}
		seeded, err := db.SeedAdmin(cfg.InitialAdminUser, hash)
		if err != nil {
			log.Printf("SeedAdmin: %v", err)
		}
		if seeded {
			log.Printf("Initial admin user '%s' seeded", cfg.InitialAdminUser)
		}
		if cfg.InitialAdminPassFile != "" {
			if err := os.Remove(cfg.InitialAdminPassFile); err != nil && !os.IsNotExist(err) {
				log.Printf("Could not remove initial admin password file %s: %v", cfg.InitialAdminPassFile, err)
			}
		}
	}

	// Seed secret_path in settings if not already set
	if sp := db.GetSetting("secret_path", ""); sp == "" {
		db.SetSetting("secret_path", cfg.SecretPath)
	}

	// ── Background metrics collector ───────────────────────────────────────────
	collector := monitoring.NewCollector()
	collector.Start(time.Duration(cfg.MetricsInterval) * time.Second)

	// ── WebSocket hub ──────────────────────────────────────────────────────────
	hub := appws.NewHub()
	go hub.Run()

	go func() {
		ticker := time.NewTicker(time.Duration(cfg.MetricsInterval) * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			hub.Broadcast("system.metrics", collector.Latest())
		}
	}()

	if cfg.SMTPPass == "" {
		if cfg.SMTPPassFile != "" {
			if _, err := os.Stat(cfg.SMTPPassFile); err == nil {
				raw, readErr := os.ReadFile(cfg.SMTPPassFile)
				if readErr != nil {
					log.Printf("SMTP password file read failed (%s): %v", cfg.SMTPPassFile, readErr)
				} else {
					cfg.SMTPPass = strings.TrimSpace(string(raw))
					if cfg.SMTPPass != "" {
						if err := db.SetSecretSetting("smtp_pass", cfg.SMTPPass); err != nil {
							log.Printf("could not persist encrypted SMTP password in DB: %v", err)
						}
					}
				}
				if err := os.Remove(cfg.SMTPPassFile); err != nil && !os.IsNotExist(err) {
					log.Printf("Could not remove SMTP password file %s: %v", cfg.SMTPPassFile, err)
				}
			}
		}

		// DB encrypted value overrides bootstrap/env when available.
		if dbPass := db.GetSecretSetting("smtp_pass", ""); dbPass != "" {
			cfg.SMTPPass = dbPass
		}
	}

	// ── Mailer ─────────────────────────────────────────────────────────────────
	mailer := notify.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.AlertEmail)

	// ── HTTP handlers ──────────────────────────────────────────────────────────
	h := api.NewHandlers(cfg, collector, hub, mailer)
	h.InitGrantStore()
	h.StartTaskScheduler()

	// ── Start periodic alert ingestion ────────────────────────────────────────
	go func() {
		api.IngestSystemAlerts()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			api.IngestSystemAlerts()
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(api.SecurityHeaders)

	// ── Secret link gate (wraps EVERYTHING below) ──────────────────────────────
	r.Use(api.GateMiddleware(cfg.JWTSecret))

	// ── Public auth routes (inside gate, no JWT needed) ────────────────────────
	r.Post("/api/v1/auth/login", h.Login)
	r.Post("/api/v1/auth/logout", h.Logout)
	r.Post("/api/v1/auth/2fa/verify", h.Verify2FA)

	// ── Terminal WebSocket — no request timeout (long-lived connection) ────────
	r.With(api.RequireAuth(cfg.JWTSecret)).Get("/api/v1/terminal/ws", h.TerminalWS)
	r.With(api.RequireAuth(cfg.JWTSecret)).Get("/api/v1/docker/containers/{id}/logs/stream", h.ContainerLogsWS)

	// ── All authenticated routes (60s timeout) ────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * time.Second))
		r.Use(api.RequireAuth(cfg.JWTSecret))
		r.Use(api.CSRFProtect())

		// Own profile & 2FA (every authenticated user)
		r.Get("/api/v1/me", h.GetMe)
		r.Get("/api/v1/auth/2fa/setup", h.Setup2FA)
		r.Post("/api/v1/auth/2fa/enable", h.Enable2FA)
		r.Delete("/api/v1/auth/2fa/disable", h.Disable2FA)

		// Lock screen PIN (own user only)
		r.Get("/api/v1/lock/settings", h.GetLockSettings)
		r.Post("/api/v1/lock/pin", h.SaveLockPin)
		r.Delete("/api/v1/lock/pin", h.ClearLockPin)
		r.Post("/api/v1/lock/verify", h.VerifyLockPin)

		// Read-only system data (all authenticated users)
		r.Get("/api/v1/system/metrics", h.GetMetrics)
		r.Get("/api/v1/system/processes", h.GetProcesses)
		r.Get("/api/v1/system/services", h.GetServices)
		r.Get("/api/v1/system/suspicious", h.GetSuspiciousProcesses)
		r.Get("/api/v1/system/disk-usage", h.GetDiskUsageTree)
		r.Get("/api/v1/system/health", h.GetHealth)
		r.Post("/api/v1/system/health/fix", h.FixHealthIssue)
		r.Get("/api/v1/system/tunnelable-apps", h.GetTunnelableApps)
		r.Post("/api/v1/system/tunnelable-apps/grant", h.GrantTunnelAccess)
		r.Get("/api/v1/system/cleanup/stats", h.GetCleanupStats)
		r.Get("/api/v1/system/cleanup/logs", h.GetCleanupLogs)
		r.Get("/api/v1/docker/info", h.GetDockerInfo)
		r.Get("/api/v1/docker/containers", h.GetDockerContainers)
		r.Get("/api/v1/docker/containers/{id}/logs", h.GetContainerLogs)
		r.Get("/api/v1/docker/containers/{id}/stats", h.GetContainerStats)
		r.Get("/api/v1/logs", h.GetLogs)
		r.Get("/api/v1/alerts", h.GetAlerts)
		r.Put("/api/v1/alerts/{id}/read", h.MarkAlertRead)
		r.Put("/api/v1/alerts/read", h.MarkAlertsRead)
		r.Get("/api/v1/audit-logs", h.GetAuditLogs)
		r.Get("/api/v1/dashboard/login-attempts", h.GetDashboardLoginAttempts)
		r.Get("/api/v1/dashboard/layout", h.GetDashboardLayout)
		r.Put("/api/v1/dashboard/layout", h.SaveDashboardLayout)
		r.Get("/api/v1/security/status", h.GetSecurityStatus)
		r.Get("/api/v1/security/bans", h.GetBans)
		r.Get("/api/v1/security-tools", h.GetSecurityTools)
		r.Get("/api/v1/security-tools/{name}/logs", h.GetSecurityToolLogs)
		r.Get("/api/v1/firewall/rules", h.GetFirewallRules)
		r.Get("/api/v1/services", h.GetManagedServices)
		r.Get("/api/v1/services/install/logs", h.GetServiceInstallLogs)
		r.Get("/api/v1/services/config", h.GetServiceConfig)
		// Apps
		r.Get("/api/v1/apps", h.GetApps)
		r.Get("/api/v1/apps/op/logs", h.GetAppOpLogs)
		r.Get("/api/v1/tasks", h.GetTasksV2)
		r.Get("/api/v1/updates", h.GetUpdates)
		r.Get("/api/v1/updates/logs", h.GetUpdateLogs)
		r.Get("/api/v1/settings", h.GetSettings)
	})

	// ── Admin-only routes (60s timeout, admin role required) ──────────────────
	r.Group(func(r chi.Router) {
		r.Use(middleware.Timeout(60 * time.Second))
		r.Use(api.RequireAuth(cfg.JWTSecret))
		r.Use(api.RequireRole("admin", "superadmin"))

		// Settings mutations
		r.Put("/api/v1/settings", h.UpdateSettings)

		// Security mutations
		r.Delete("/api/v1/security/bans/{ip}", h.Unban)
		r.Post("/api/v1/security/bans/{ip}", h.BanIP)
		r.Post("/api/v1/security-tools/{name}/run", h.RunSecurityTool)
		r.Post("/api/v1/security-tools/{name}/install", h.InstallSecurityTool)

		// Firewall mutations
		r.Post("/api/v1/firewall/rules", h.AddFirewallRule)
		r.Delete("/api/v1/firewall/rules/{id}", h.DeleteFirewallRule)

		// Docker mutations
		r.Post("/api/v1/docker/prune/{kind}", h.DockerPrune)
		r.Post("/api/v1/containers/{id}/start", h.ContainerStart)
		r.Post("/api/v1/containers/{id}/stop", h.ContainerStop)
		r.Post("/api/v1/containers/{id}/restart", h.ContainerRestart)

		// User management
		r.Get("/api/v1/users", h.GetUsers)
		r.Post("/api/v1/users", h.CreateUser)
		r.Put("/api/v1/users/{id}", h.UpdateUser)
		r.Delete("/api/v1/users/{id}", h.DeleteUser)

		// Updates
		r.Post("/api/v1/updates/install", h.InstallUpdates)
		r.Post("/api/v1/system/cleanup/run", h.RunCleanup)

		// Services mutations
		r.Post("/api/v1/services/{name}/install", h.ServiceInstall)
		r.Post("/api/v1/services/{name}/{action}", h.ServiceAction)
		r.Put("/api/v1/services/config", h.UpdateServiceConfig)

		// Apps mutations
		r.Post("/api/v1/apps/{name}/install", h.InstallApp)
		r.Post("/api/v1/apps/{name}/update", h.UpdateApp)
		r.Delete("/api/v1/apps/{name}", h.UninstallApp)

		// Config file editor
		r.Get("/api/v1/services/{name}/configfile", h.GetServiceConfigFile)
		r.Put("/api/v1/services/{name}/configfile", h.SaveServiceConfigFile)
		r.Post("/api/v1/services/{name}/configfile/backup", h.BackupServiceConfigFile)
		r.Post("/api/v1/services/{name}/configfile/verify", h.VerifyServiceConfigFile)
		r.Post("/api/v1/services/{name}/configfile/restore", h.RestoreServiceConfigFile)

		// Tasks mutations
		r.Post("/api/v1/tasks", h.CreateTaskV2)
		r.Put("/api/v1/tasks/{id}", h.UpdateTaskV2)
		r.Delete("/api/v1/tasks/{id}", h.DeleteTaskV2)
		r.Post("/api/v1/tasks/{id}/run", h.RunTaskNow)

		// DB admin
		r.Get("/api/v1/db/stats", h.GetDBStats)
		r.Get("/api/v1/db/export", h.ExportDB)
		r.Post("/api/v1/db/import", h.ImportDB)
		r.Post("/api/v1/db/prune", h.PruneDB)
	})

	// ── WebSocket (metrics) ────────────────────────────────────────────────────
	r.Get("/ws", h.WSConnect)

	// ── Static frontend (SPA) ─────────────────────────────────────────────────
	frontendDir := cfg.FrontendDir
	if _, err := os.Stat(frontendDir); err != nil {
		frontendDir = "./frontend/dist"
	}
	r.Handle("/*", spaHandler(frontendDir))

	// ── HTTP server ────────────────────────────────────────────────────────────
	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("[sentinelcore] Listening on http://%s  |  secret path: /%s", cfg.ListenAddr, cfg.SecretPath)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[sentinelcore] Shutting down…")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
}

func jsonOK(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// spaHandler serves static files from dir; falls back to index.html for any
// path that is not an existing file (so Vue Router handles client-side routes).
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fullPath := filepath.Join(dir, filepath.Clean(r.URL.Path))
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}

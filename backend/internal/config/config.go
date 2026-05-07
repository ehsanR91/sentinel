package config

import (
	"os"
	"strconv"
)

type Config struct {
	ListenAddr     string
	FrontendDir    string
	JWTSecret      string
	DBPath         string
	SecretsKeyPath string
	LogLevel       string

	MetricsInterval int

	WatchedServices []string

	// Secret link gate
	SecretPath string

	// Initial admin seeding (consumed once at first start)
	InitialAdminUser     string
	InitialAdminPass     string
	InitialAdminPassFile string

	// SMTP / alerting
	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPass     string
	SMTPPassFile string
	AlertEmail   string

	// Brute force protection
	BruteForceThreshold int
}

func Load() *Config {
	return &Config{
		ListenAddr:      getEnv("LISTEN_ADDR", "127.0.0.1:8080"),
		FrontendDir:     getEnv("FRONTEND_DIR", "./frontend/dist"),
		JWTSecret:       getEnv("JWT_SECRET", "change-me-in-production"),
		DBPath:          getEnv("DB_PATH", "./data/app.db"),
		SecretsKeyPath:  getEnv("SECRETS_KEY_PATH", ""),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		MetricsInterval: getEnvInt("METRICS_INTERVAL", 2),
		SecretPath:      getEnv("SECRET_PATH", "sentinel-core"),

		InitialAdminUser:     getEnv("INITIAL_ADMIN_USERNAME", "admin"),
		InitialAdminPass:     getEnv("INITIAL_ADMIN_PASSWORD", ""),
		InitialAdminPassFile: getEnv("INITIAL_ADMIN_PASSWORD_FILE", ""),

		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPass:     getEnv("SMTP_PASS", ""),
		SMTPPassFile: getEnv("SMTP_PASS_FILE", ""),
		AlertEmail:   getEnv("ALERT_EMAIL", ""),

		BruteForceThreshold: getEnvInt("BRUTE_FORCE_THRESHOLD", 5),

		WatchedServices: []string{
			"ufw", "fail2ban", "crowdsec", "psad", "clamav-daemon",
			"auditd", "apparmor", "docker", "netdata", "unattended-upgrades",
			"aide", "rkhunter", "nginx", "sshd",
		},
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

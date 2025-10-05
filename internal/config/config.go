package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                       string
	DSN                        string
	JWT                        string
	SMTPHost                   string
	SMTPPort                   string
	SMTPUsername               string
	SMTPPassword               string
	SMTPFrom                   string
	AQIBaseURL                 string
	NotificationIntervalMinute int
	MLServiceURL               string
	MLPredictPath              string
}

func Load() Config {
	_ = godotenv.Load()

	return Config{
		Port:                       env("PORT", "8080"),
		DSN:                        buildDSN(),
		JWT:                        env("JWT_SECRET", "super-secret-change-me"),
		SMTPHost:                   env("SMTP_HOST", ""),
		SMTPPort:                   env("SMTP_PORT", ""),
		SMTPUsername:               env("SMTP_USERNAME", ""),
		SMTPPassword:               env("SMTP_PASSWORD", ""),
		SMTPFrom:                   env("SMTP_FROM", ""),
		AQIBaseURL:                 env("AQI_BASE_URL", ""),
		NotificationIntervalMinute: envInt("NOTIFICATION_INTERVAL_MIN", 30),
		MLServiceURL:               env("ML_SERVICE_URL", ""),
		MLPredictPath:              env("ML_PREDICT_PATH", ""),
	}
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func buildDSN() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	host := env("DB_HOST", "localhost")
	port := env("DB_PORT", "5432")
	user := env("DB_USERNAME", "postgres")
	pass := env("DB_PASSWORD", "password")
	db := env("DB_DATABASE", "rlaas")
	sch := env("DB_SCHEMA", "public")

	return "postgres://" + user + ":" + pass + "@" + host + ":" + port +
		"/" + db + "?sslmode=disable&search_path=" + sch
}

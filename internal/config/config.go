package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresUser     string
	PostgresPass     string
	PostgresDB       string
	PostgresHost     string
	PostgresPort     string
	AppPort          string
	JWTSecret        string
	FrontendURI      string
	OAuthRedirectURL string
}

type OauthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

func LoadConfig(path string) (*Config, error) {
	_ = godotenv.Load(path)

	// PORT öncelikli (Heroku için), yoksa APP_PORT
	appPort := os.Getenv("PORT")
	if appPort == "" {
		appPort = os.Getenv("APP_PORT")
	}
	if appPort == "" {
		appPort = "8080"
	}

	cfg := &Config{
		PostgresUser:     os.Getenv("POSTGRES_USER"),
		PostgresPass:     os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:       os.Getenv("POSTGRES_DB"),
		PostgresHost:     os.Getenv("POSTGRES_HOST"),
		PostgresPort:     os.Getenv("POSTGRES_PORT"),
		AppPort:          appPort,
		JWTSecret:        os.Getenv("JWT_SECRET"),
		FrontendURI:      os.Getenv("FRONTEND_URI"),
		OAuthRedirectURL: os.Getenv("OAUTH_REDIRECT_URL"),
	}

	return cfg, nil
}

func LoadOauthConfig(path string) (*OauthConfig, error) {
	_ = godotenv.Load(path)
	cfg := &OauthConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("OAUTH_REDIRECT_URL"),
	}
	return cfg, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func BuildDSN() string {
	// DATABASE_URL varsa onu kullan (Heroku için)
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	// Local fallback
	host := env("POSTGRES_HOST", "localhost")
	port := env("POSTGRES_PORT", "5432")
	user := env("POSTGRES_USER", "postgres")
	pass := env("POSTGRES_PASSWORD", "postgres")
	db := env("POSTGRES_DB", "nasa")

	return "postgres://" + user + ":" + pass + "@" + host + ":" + port +
		"/" + db + "?sslmode=disable"
}

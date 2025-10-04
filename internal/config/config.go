package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	PostgresUser string
	PostgresPass string
	PostgresDB   string
	PostgresHost string
	PostgresPort string
	AppPort      string
	JWTSecret    string
	FrontendURI  string
}

type OauthConfig struct {
	ClientID     string // Public olmalı (büyük harfle başlamalı)
	ClientSecret string // Public olmalı
}

func LoadConfig(path string) (*Config, error) {
	_ = godotenv.Load(path)

	cfg := &Config{
		PostgresUser: os.Getenv("POSTGRES_USER"),
		PostgresPass: os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:   os.Getenv("POSTGRES_DB"),
		PostgresHost: os.Getenv("POSTGRES_HOST"),
		PostgresPort: os.Getenv("POSTGRES_PORT"),
		AppPort:      os.Getenv("APP_PORT"),
		JWTSecret:    os.Getenv("JWT_SECRET"),
		FrontendURI:  os.Getenv("FRONTEND_URI"),
	}

	return cfg, nil
}

func LoadOauthConfig(path string) (*OauthConfig, error) {
	_ = godotenv.Load(path)
	cfg := &OauthConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
	}
	return cfg, nil
}

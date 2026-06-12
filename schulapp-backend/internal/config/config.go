package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	JWTSecret   string
	Port        string
	UploadDir   string
	FrontendURL string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET env var is required")
	}

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/schulapp?sslmode=disable"),
		JWTSecret:   secret,
		Port:        getEnv("PORT", "8080"),
		UploadDir:   getEnv("UPLOAD_DIR", "./uploads"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

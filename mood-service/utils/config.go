package utils

import (
	"log"
	"os"
)

type Config struct {
	PostgresDSN    string // e.g. postgres://postgres:password@postgres:5432/moodtracker?sslmode=disable
	AuthServiceURL string // e.g. http://auth-service:8081
	JWTSecret      string // same secret as auth-service
	Port           string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		PostgresDSN:    os.Getenv("POSTGRES_DSN"),
		AuthServiceURL: os.Getenv("AUTH_SERVICE_URL"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		Port:           os.Getenv("PORT"),
	}
	if cfg.Port == "" {
		cfg.Port = "8082"
	}
	log.Printf("[mood-service] loaded config: %+v\n", cfg)
	return cfg, nil
}

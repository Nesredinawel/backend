package utils

import (
	"log"
	"os"
)

type Config struct {
	HasuraEndpoint     string
	HasuraAdminSecret  string
	JWTSecret          string
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	Port               string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		HasuraEndpoint:     os.Getenv("HASURA_GRAPHQL_ENDPOINT"),
		HasuraAdminSecret:  os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Port:               os.Getenv("PORT"),
	}

	if cfg.HasuraEndpoint == "" || cfg.HasuraAdminSecret == "" {
		log.Println("⚠️ WARNING: Hasura endpoint or admin secret missing")
	}
	if cfg.Port == "" {
		cfg.Port = "8081"
	}

	log.Printf("[DEBUG] Loaded config: %+v\n", cfg)
	return cfg, nil
}

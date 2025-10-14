package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	HasuraURL          string
	HasuraAdminSecret  string
	JWTSecret          string
}

func LoadConfig() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		HasuraURL:          os.Getenv("HASURA_GRAPHQL_ENDPOINT"),
		HasuraAdminSecret:  os.Getenv("HASURA_ADMIN_SECRET"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
	}

	if cfg.GoogleClientID == "" || cfg.GoogleClientSecret == "" {
		log.Fatal("missing Google OAuth credentials in environment")
	}

	return cfg, nil
}

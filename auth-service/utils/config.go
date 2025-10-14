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
	HasuraEndpoint     string
	HasuraAdminSecret  string
	JWTSecret          string
}

func LoadConfig() Config {
	// try to load .env but continue if not present
	err := godotenv.Load()
	if err != nil {
		log.Println("no .env file found, reading environment variables")
	}
	return Config{
		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8081/auth/google/callback"),
		HasuraEndpoint:     getEnv("HASURA_GRAPHQL_ENDPOINT", "http://hasura:8080/v1/graphql"),
		HasuraAdminSecret:  getEnv("HASURA_GRAPHQL_ADMIN_SECRET", "hasura_secret"),
		JWTSecret:          getEnv("JWT_SECRET", "supersecretkey"),
	}
}

func getEnv(key, fallback string) string {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v
}

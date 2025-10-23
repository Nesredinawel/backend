package utils

import (
	"log"
	"os"
)

type Config struct {
	HasuraEndpoint    string
	HasuraAdminSecret string
	JWTSecret         string
	Port              string
}

func LoadConfig() (Config, error) {
	cfg := Config{
		HasuraEndpoint:    os.Getenv("HASURA_GRAPHQL_ENDPOINT"),
		HasuraAdminSecret: os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET"),
		JWTSecret:         os.Getenv("JWT_SECRET"), // same as auth-service
		Port:              os.Getenv("PORT_2"),     // default mood-service port
	}

	if cfg.HasuraEndpoint == "" || cfg.HasuraAdminSecret == "" {
		log.Println("⚠️ WARNING: Hasura endpoint or admin secret missing")
	}

	if cfg.Port == "" {
		cfg.Port = "8082"
	}

	log.Printf("[mood-service] Loaded config: %+v\n", cfg)
	return cfg, nil
}

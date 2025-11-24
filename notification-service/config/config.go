package config

import "os"

type Config struct {
	// Core
	Port      string
	JWTSecret string

	// Redis
	RedisAddr string

	// Hasura
	HasuraEndpoint    string
	HasuraAdminSecret string
	HasuraSchema      string

	// Firebase
	EnablePush            bool
	GoogleCredentialsPath string
	FirebaseProjectID     string

	// Email
	EnableEmail    bool
	SendgridAPIKey string
}

func LoadConfig() Config {
	return Config{
		// Service port
		Port: getenv("PORT", "8084"),

		// Shared JWT secret
		JWTSecret: getenv("JWT_SECRET", ""),

		// Shared Redis address
		RedisAddr: getenv("REDIS_ADDR", "redis:6379"),

		// Shared Hasura config
		HasuraEndpoint:    getenv("HASURA_ENDPOINT", "http://hasura:8080/v1/graphql"),
		HasuraAdminSecret: getenv("HASURA_GRAPHQL_ADMIN_SECRET", ""),
		HasuraSchema:      getenv("HASURA_NOTIF_SCHEMA", "notification_service"),

		// Firebase push notifications
		EnablePush:            getenv("ENABLE_PUSH", "false") == "true",
		GoogleCredentialsPath: getenv("GOOGLE_APPLICATION_CREDENTIALS", "/keys/firebase-service-account.json"),
		FirebaseProjectID:     getenv("FIREBASE_PROJECT_ID", ""),

		// Email delivery
		EnableEmail:    getenv("ENABLE_EMAIL", "false") == "true",
		SendgridAPIKey: getenv("SENDGRID_API_KEY", ""),
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

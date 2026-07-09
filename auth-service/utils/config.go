package utils

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// 🗄️ Hasura
	HasuraEndpoint    string
	HasuraAdminSecret string

	// 🔐 JWT
	JWTSecret string

	// 🌐 Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	// 💌 Email Verification (Resend / Brevo)
	ResendAPIKey  string
	PublicBaseURL string

	// 🧠 Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// ⚙️ Service Port
	Port string
}

func LoadConfig() (Config, error) {
	dbIndex := 0
	if envDB := os.Getenv("REDIS_DB"); envDB != "" {
		if v, err := strconv.Atoi(envDB); err == nil {
			dbIndex = v
		}
	}

	cfg := Config{
		HasuraEndpoint:    os.Getenv("HASURA_GRAPHQL_ENDPOINT"),
		HasuraAdminSecret: os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET"),

		JWTSecret: os.Getenv("JWT_SECRET"),

		GoogleClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleRedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),

		ResendAPIKey:  os.Getenv("RESEND_API_KEY"),
		PublicBaseURL: os.Getenv("PUBLIC_BASE_URL"),

		RedisAddr:     os.Getenv("REDIS_ADDR"),
		RedisPassword: os.Getenv("REDIS_PASSWORD"),
		RedisDB:       dbIndex,

		Port: os.Getenv("PORT_1"),
	}

	// Default fallbacks
	if cfg.Port == "" {
		cfg.Port = "8081"
	}
	// utils/config.go (inside LoadConfig)
	cfg.RedisAddr = os.Getenv("REDIS_ADDR")
	if cfg.RedisAddr == "" || cfg.RedisAddr == "localhost:6379" {
		log.Println("⚠️ REDIS_ADDR is localhost inside Docker, switching to service name 'redis:6379'")
		cfg.RedisAddr = "redis:6379"
	}

	// 🚀 Auto-fix Redis addr for Docker if localhost is set
	if strings.HasPrefix(cfg.RedisAddr, "localhost") || strings.HasPrefix(cfg.RedisAddr, "127.0.0.1") {
		log.Println("⚠️ Detected REDIS_ADDR points to localhost. Switching to Docker service 'redis:6379'")
		cfg.RedisAddr = "redis:6379"
	}

	if cfg.PublicBaseURL == "" {
		cfg.PublicBaseURL = "http://auth-service:" + cfg.Port
	}

	// Debug warnings
	if cfg.HasuraEndpoint == "" || cfg.HasuraAdminSecret == "" {
		log.Println("⚠️ WARNING: Missing Hasura endpoint or admin secret")
	}
	if cfg.ResendAPIKey == "" {
		log.Println("⚠️ WARNING: Missing RESEND_API_KEY (email verification won’t work)")
	}

	log.Printf("[DEBUG] Loaded config: %+v\n", cfg)
	return cfg, nil
}

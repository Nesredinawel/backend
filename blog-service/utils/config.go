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

	// 🧠 Redis
	RedisAddr     string
	RedisPassword string
	RedisDB       int

	// ⚙️ Service Port
	Port string
}

// LoadConfig loads environment variables and initializes Redis automatically
func LoadConfig() Config {
	dbIndex := 0
	if envDB := os.Getenv("REDIS_DB"); envDB != "" {
		if v, err := strconv.Atoi(envDB); err == nil {
			dbIndex = v
		}
	}

	cfg := Config{
		HasuraEndpoint:    os.Getenv("HASURA_GRAPHQL_ENDPOINT"),
		HasuraAdminSecret: os.Getenv("HASURA_GRAPHQL_ADMIN_SECRET"),
		JWTSecret:         os.Getenv("JWT_SECRET"),
		Port:              os.Getenv("PORT_3"),
		RedisAddr:         os.Getenv("REDIS_ADDR"),
		RedisPassword:     os.Getenv("REDIS_PASSWORD"),
		RedisDB:           dbIndex,
	}

	// Defaults
	if cfg.Port == "" {
		cfg.Port = "8083"
	}
	if cfg.RedisAddr == "" || strings.HasPrefix(cfg.RedisAddr, "localhost") {
		log.Println("⚠️ Redis address missing or localhost, switching to Docker service 'redis:6379'")
		cfg.RedisAddr = "redis:6379"
	}

	log.Printf("[mood-service] Loaded config: %+v\n", cfg)

	// Initialize global Redis automatically
	InitRedis(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB)

	return cfg
}

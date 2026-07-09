package utils

import (
	"context"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

func InitRedis(cfg Config) {
	maxRetries := 5
	backoff := time.Second * 1

	log.Printf("[DEBUG] Attempting to connect to Redis at %s (DB=%d)\n", cfg.RedisAddr, cfg.RedisDB)

	for i := 1; i <= maxRetries; i++ {
		Rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		pong, err := Rdb.Ping(Ctx).Result()
		if err == nil {
			log.Printf("Connected to Redis at %s (pong=%s)\n", cfg.RedisAddr, pong)
			return
		}

		log.Printf("Attempt %d/%d: Failed to connect to Redis (%v). Retrying in %v...\n",
			i, maxRetries, err, backoff)

		time.Sleep(backoff)
		backoff *= 2
	}

	log.Printf("WARNING: Could not connect to Redis after %d attempts. Running without Redis (email verification and rate limiting degraded).\n", maxRetries)
	Rdb = nil
}

package utils

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	Rdb      *redis.Client
	initOnce sync.Once
)

// InitRedis initializes a global Redis client (safe to call multiple times)
func InitRedis(addr, password string, db int) *redis.Client {
	initOnce.Do(func() {
		Rdb = redis.NewClient(&redis.Options{
			Addr:     addr,
			Password: password,
			DB:       db,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if _, err := Rdb.Ping(ctx).Result(); err != nil {
			log.Printf("⚠️ Redis unavailable (%v). Running without Redis — notifications disabled.", err)
			Rdb = nil
			return
		}

		log.Println("✅ Redis initialized")
	})
	return Rdb
}

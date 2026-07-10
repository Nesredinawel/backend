package utils

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	Rdb *redis.Client
	Ctx = context.Background()
)

func InitRedis(addr string) {
	Rdb = redis.NewClient(&redis.Options{
		Addr: addr,
	})
	if err := Rdb.Ping(Ctx).Err(); err != nil {
		log.Printf("⚠️ Redis unavailable (%v). Running without Redis — notifications disabled.", err)
		Rdb = nil
		return
	}
	log.Println("✅ Connected to Redis:", addr)
}

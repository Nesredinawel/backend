// utils/redis.go
package utils

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type PendingSignup struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

// InitRedisClient initializes a Redis client with retry/backoff and detailed debug logs
func InitRedisClient(cfg Config) *redis.Client {
	var rdb *redis.Client
	var err error

	ctx := context.Background()
	maxRetries := 10
	backoff := time.Second * 1

	log.Printf("[DEBUG] Attempting to connect to Redis at %s (DB=%d)\n", cfg.RedisAddr, cfg.RedisDB)

	for i := 1; i <= maxRetries; i++ {
		rdb = redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		pong, err := rdb.Ping(ctx).Result()
		if err == nil {
			log.Printf("✅ Connected to Redis at %s (pong=%s)\n", cfg.RedisAddr, pong)
			return rdb
		}

		log.Printf("⚠️ Attempt %d/%d: Failed to connect to Redis (%v). Retrying in %v...\n", i, maxRetries, err, backoff)
		time.Sleep(backoff)
		backoff *= 2
	}

	panic(fmt.Sprintf("❌ Could not connect to Redis after %d attempts: %v", maxRetries, err))
}

// GenerateVerificationToken creates a secure random token with debug
func GenerateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Println("❌ Failed to generate verification token:", err)
		return "", err
	}
	token := hex.EncodeToString(b)
	log.Println("🔑 Generated verification token:", token)
	return token, nil
}

// SavePendingSignup stores signup data in Redis temporarily with detailed debug
func SavePendingSignup(rdb *redis.Client, token string, data PendingSignup, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("verify:%s", token)

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("❌ Failed to marshal pending signup data:", err)
		return err
	}

	log.Printf("[DEBUG] Saving pending signup to Redis: key=%s, email=%s, ttl=%s\n", key, data.Email, ttl)

	if err := rdb.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		log.Printf("❌ Failed to save pending signup in Redis: key=%s, err=%v\n", key, err)
		return err
	}

	log.Printf("💾 Successfully saved pending signup in Redis: key=%s, email=%s\n", key, data.Email)
	return nil
}

// GetPendingSignup retrieves signup data from Redis with debug
func GetPendingSignup(rdb *redis.Client, token string) (PendingSignup, error) {
	ctx := context.Background()
	key := fmt.Sprintf("verify:%s", token)

	log.Printf("[DEBUG] Retrieving pending signup from Redis: key=%s\n", key)

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		log.Printf("❌ Failed to get pending signup from Redis: key=%s, err=%v\n", key, err)
		return PendingSignup{}, err
	}

	var data PendingSignup
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		log.Printf("❌ Failed to unmarshal pending signup JSON: key=%s, err=%v\n", key, err)
		return PendingSignup{}, err
	}

	log.Printf("📥 Successfully retrieved pending signup from Redis: key=%s, email=%s\n", key, data.Email)
	return data, nil
}

// DeletePendingSignup removes signup data after verification with debug
func DeletePendingSignup(rdb *redis.Client, token string) {
	ctx := context.Background()
	key := fmt.Sprintf("verify:%s", token)

	log.Printf("[DEBUG] Deleting pending signup from Redis: key=%s\n", key)

	if err := rdb.Del(ctx, key).Err(); err != nil {
		log.Printf("❌ Failed to delete pending signup from Redis: key=%s, err=%v\n", key, err)
	} else {
		log.Printf("🗑️ Successfully deleted pending signup from Redis: key=%s\n", key)
	}
}

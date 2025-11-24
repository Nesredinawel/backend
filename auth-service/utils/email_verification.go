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

// GenerateVerificationToken creates a secure random token
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

// SavePendingSignup stores signup data in Redis temporarily
func SavePendingSignup(rdb *redis.Client, token string, data PendingSignup, ttl time.Duration) error {
	ctx := context.Background()
	key := fmt.Sprintf("verify:%s", token)

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Println("❌ Failed to marshal pending signup data:", err)
		return err
	}

	if err := rdb.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		log.Printf("❌ Failed to save pending signup in Redis: %v\n", err)
		return err
	}

	return nil
}

// GetPendingSignup retrieves signup data
func GetPendingSignup(rdb *redis.Client, token string) (PendingSignup, error) {
	ctx := context.Background()
	key := fmt.Sprintf("verify:%s", token)

	val, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return PendingSignup{}, err
	}

	var data PendingSignup
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return PendingSignup{}, err
	}

	return data, nil
}

// DeletePendingSignup removes signup data
func DeletePendingSignup(rdb *redis.Client, token string) {
	ctx := context.Background()
	key := fmt.Sprintf("verify:%s", token)
	rdb.Del(ctx, key)
}

package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"
)

type PendingSignup struct {
	Name         string `json:"name"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
}

var (
	pendingSignupsMu sync.RWMutex
	pendingSignups   = map[string]struct {
		data PendingSignup
		exp  time.Time
	}{}
)

// GenerateVerificationToken creates a secure random token
func GenerateVerificationToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		log.Println("❌ Failed to generate verification token:", err)
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SavePendingSignup stores signup data in memory temporarily
func SavePendingSignup(rdb interface{}, token string, data PendingSignup, ttl time.Duration) error {
	pendingSignupsMu.Lock()
	pendingSignups[token] = struct {
		data PendingSignup
		exp  time.Time
	}{data: data, exp: time.Now().Add(ttl)}
	pendingSignupsMu.Unlock()
	return nil
}

// GetPendingSignup retrieves signup data
func GetPendingSignup(rdb interface{}, token string) (PendingSignup, error) {
	pendingSignupsMu.RLock()
	entry, ok := pendingSignups[token]
	pendingSignupsMu.RUnlock()

	if !ok {
		return PendingSignup{}, fmt.Errorf("verification token not found or expired")
	}

	if time.Now().After(entry.exp) {
		pendingSignupsMu.Lock()
		delete(pendingSignups, token)
		pendingSignupsMu.Unlock()
		return PendingSignup{}, fmt.Errorf("verification token expired")
	}

	return entry.data, nil
}

// DeletePendingSignup removes signup data
func DeletePendingSignup(rdb interface{}, token string) {
	pendingSignupsMu.Lock()
	delete(pendingSignups, token)
	pendingSignupsMu.Unlock()
}

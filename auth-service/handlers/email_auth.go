package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"auth-service/models"
	"auth-service/utils"

	"github.com/redis/go-redis/v9"
)

// Redis client (initialized with Docker service name)
var rdb = redis.NewClient(&redis.Options{
	Addr: "redis:6379", // Docker service name in docker-compose
})

// ===========================================
// 📧 STEP 1: Signup → Send verification email or add password to Google user
// ===========================================
func EmailSignup(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		// 🔍 Check if user already exists
		existingUser, err := utils.GetUserByEmail(cfg, req.Email)
		if err == nil && existingUser.ID != "" {
			// ✅ Case: existing Google user adding password (local login)
			if existingUser.Provider == "google" && existingUser.Password == "" {
				hash, err := utils.HashPassword(req.Password)
				if err != nil {
					http.Error(w, "failed to hash password", http.StatusInternalServerError)
					return
				}

				// Update password & provider to support both logins
				_, err = utils.UpdateUserPasswordAndProvider(cfg, existingUser.ID, hash, "local")
				if err != nil {
					log.Printf("❌ failed to update password for Google user: %v", err)
					http.Error(w, "failed to update password", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"message": "✅ Password added successfully. You can now log in with Google or email.",
				})
				return
			}

			// 🚫 User already exists (not upgradable)
			http.Error(w, "user already exists", http.StatusConflict)
			return
		}

		// 🆕 New signup flow → send verification email
		hash, err := utils.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		token, err := utils.GenerateVerificationToken()
		if err != nil {
			http.Error(w, "failed to create token", http.StatusInternalServerError)
			return
		}

		// Save pending signup temporarily in Redis
		pending := utils.PendingSignup{
			Name:         req.Name,
			Email:        req.Email,
			PasswordHash: hash,
		}
		if err := utils.SavePendingSignup(rdb, token, pending, 15*time.Minute); err != nil {
			http.Error(w, "failed to store verification data", http.StatusInternalServerError)
			return
		}

		verifyURL := fmt.Sprintf("%s/auth/email/verify?token=%s", cfg.PublicBaseURL, token)
		log.Println("🔗 Verification URL:", verifyURL)

		// Send verification email
		if err := utils.SendVerificationEmail(cfg.ResendAPIKey, req.Email, verifyURL); err != nil {
			log.Printf("❌ Failed to send verification email: %v", err)
			http.Error(w, "failed to send email", http.StatusInternalServerError)
			return
		}

		resp := map[string]string{
			"message": "✅ Check your email to confirm signup. Link valid for 15 minutes.",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ===========================================
// 📨 STEP 2: Email verify → Create user or link to existing Google user
// ===========================================
func EmailVerify(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}

		// Retrieve pending signup
		pending, err := utils.GetPendingSignup(rdb, token)
		if err != nil {
			http.Error(w, "invalid or expired verification token", http.StatusUnauthorized)
			return
		}
		utils.DeletePendingSignup(rdb, token)

		// 🔍 Check if user already exists (e.g., signed up via Google)
		existingUser, err := utils.GetUserByEmail(cfg, pending.Email)
		if err == nil && existingUser.ID != "" {
			if existingUser.Provider == "google" && existingUser.Password == "" {
				// Upgrade Google user with password
				_, err := utils.UpdateUserPasswordAndProvider(cfg, existingUser.ID, pending.PasswordHash, "local")
				if err != nil {
					http.Error(w, "failed to link password to Google user", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]string{
					"message": "🎉 Email verified! Password linked to your Google account.",
					"user_id": existingUser.ID,
				})
				return
			}
		}

		// ✨ Otherwise, create a new user
		user := models.User{
			Email:    pending.Email,
			Name:     pending.Name,
			Password: pending.PasswordHash,
			Role:     "user",
			Provider: "local",
		}

		userID, err := utils.UpsertUserInHasura(cfg, user)
		if err != nil {
			log.Printf("❌ Failed to insert verified user: %v", err)
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		utils.CreateEmptyUserProfile(cfg, userID)

		resp := map[string]string{
			"message": "🎉 Email verified successfully. You can now log in.",
			"user_id": userID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ===========================================
// 🔐 STEP 3: Email login
// ===========================================
func EmailLogin(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request", http.StatusBadRequest)
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		user, err := utils.GetUserByEmail(cfg, req.Email)
		if err != nil {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		// If user came from Google but has no password
		if user.Provider == "google" && user.Password == "" {
			http.Error(w, "this account uses Google login, not email/password", http.StatusForbidden)
			return
		}

		if !utils.CheckPasswordHash(req.Password, user.Password) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		session, err := utils.GenerateJWT(cfg, user.ID, user.Role)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"access_token":  session.AccessToken,
			"refresh_token": session.RefreshToken,
			"expires_in":    session.ExpiresIn,
			"user": map[string]string{
				"user_id":  user.ID,
				"email":    user.Email,
				"name":     user.Name,
				"provider": user.Provider,
				"role":     user.Role,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

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

// Redis client (Docker service name used)
var rdb = redis.NewClient(&redis.Options{
	Addr: "redis:6379",
})

// ===========================================
// 📧 STEP 1: Signup → Send verification email or link Google user
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

			// Google user with no password → add password
			if existingUser.Provider == "google" && existingUser.Password == "" {

				hash, err := utils.HashPassword(req.Password)
				if err != nil {
					http.Error(w, "failed to hash password", http.StatusInternalServerError)
					return
				}

				_, err = utils.UpdateUserPasswordAndProvider(cfg, existingUser.ID, hash, "local")
				if err != nil {
					log.Printf("❌ failed to update password for Google user: %v", err)
					http.Error(w, "failed to update password", http.StatusInternalServerError)
					return
				}

				// 📢 Notify
				utils.PublishNotification(rdb, "auth_events", utils.NotificationEvent{
					UserID:        existingUser.ID,
					Title:         "Password Added",
					Message:       fmt.Sprintf("Password linked to Google account %s", existingUser.Email),
					SourceService: "auth-service",
					Action:        "PASSWORD_LINKED",
					Meta: map[string]interface{}{
						"email": existingUser.Email,
					},
				})

				jsonResponse(w, map[string]string{
					"message": "Password added successfully. You can now login using email or Google.",
				})
				return
			}

			http.Error(w, "user already exists", http.StatusConflict)
			return
		}

		// 🆕 New signup → email verification
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

		if err := utils.SendVerificationEmail(cfg.ResendAPIKey, req.Email, verifyURL); err != nil {
			log.Printf("❌ Failed to send verification email: %v", err)
			http.Error(w, "failed to send email", http.StatusInternalServerError)
			return
		}

		// 📢 Notify
		utils.PublishNotification(rdb, "auth_events", utils.NotificationEvent{
			UserID:        "pending",
			Title:         "Signup Started",
			Message:       fmt.Sprintf("Signup initiated for %s", req.Email),
			SourceService: "auth-service",
			Action:        "SIGNUP_INITIATED",
			Meta: map[string]interface{}{
				"email": req.Email,
			},
		})

		jsonResponse(w, map[string]string{
			"message": "Check your email to confirm signup. Link valid for 15 minutes.",
		})
	}
}

// ===========================================
// 📨 STEP 2: Email verify
// ===========================================
func EmailVerify(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		token := r.URL.Query().Get("token")
		if token == "" {
			http.Error(w, "missing token", http.StatusBadRequest)
			return
		}

		pending, err := utils.GetPendingSignup(rdb, token)
		if err != nil {
			http.Error(w, "invalid or expired verification token", http.StatusUnauthorized)
			return
		}
		utils.DeletePendingSignup(rdb, token)

		existingUser, err := utils.GetUserByEmail(cfg, pending.Email)
		if err == nil && existingUser.ID != "" {

			// Google user being upgraded to local+password
			if existingUser.Provider == "google" && existingUser.Password == "" {

				_, err := utils.UpdateUserPasswordAndProvider(cfg, existingUser.ID, pending.PasswordHash, "local")
				if err != nil {
					http.Error(w, "failed to link password", http.StatusInternalServerError)
					return
				}

				utils.PublishNotification(rdb, "auth_events", utils.NotificationEvent{
					UserID:        existingUser.ID,
					Title:         "Email Verified",
					Message:       fmt.Sprintf("Email verified and password linked for %s", pending.Email),
					SourceService: "auth-service",
					Action:        "EMAIL_VERIFIED_LINKED",
					Meta: map[string]interface{}{
						"email": pending.Email,
					},
				})

				jsonResponse(w, map[string]string{
					"message": "Email verified! Password linked to Google account.",
					"user_id": existingUser.ID,
				})
				return
			}
		}

		// Create new user
		user := models.User{
			Email:    pending.Email,
			Name:     pending.Name,
			Password: pending.PasswordHash,
			Role:     "user",
			Provider: "local",
		}

		userID, err := utils.UpsertUserInHasura(cfg, user)
		if err != nil {
			log.Printf("❌ Failed to insert user: %v", err)
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		utils.CreateEmptyUserProfile(cfg, userID)

		utils.PublishNotification(rdb, "auth_events", utils.NotificationEvent{
			UserID:        userID,
			Title:         "Email Verified",
			Message:       fmt.Sprintf("Email verified successfully for %s", pending.Email),
			SourceService: "auth-service",
			Action:        "EMAIL_VERIFIED",
			Meta: map[string]interface{}{
				"email": pending.Email,
			},
		})

		jsonResponse(w, map[string]string{
			"message": "Email verified successfully.",
			"user_id": userID,
		})
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

		if user.Provider == "google" && user.Password == "" {
			http.Error(w, "this account uses Google login", http.StatusForbidden)
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

		// Notify login
		utils.PublishNotification(rdb, "auth_events", utils.NotificationEvent{
			UserID:        user.ID,
			Title:         "Login Successful",
			Message:       fmt.Sprintf("%s logged in successfully", user.Email),
			SourceService: "auth-service",
			Action:        "USER_LOGIN",
			Meta: map[string]interface{}{
				"email":    user.Email,
				"provider": user.Provider,
			},
		})

		jsonResponse(w, map[string]interface{}{
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
		})
	}
}

// Helper JSON writer
func jsonResponse(w http.ResponseWriter, body interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

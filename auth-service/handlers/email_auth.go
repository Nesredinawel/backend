package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"auth-service/models"
	"auth-service/utils"
)

func EmailSignup(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "Invalid request body. Please provide valid JSON.")
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		if req.Name == "" || req.Email == "" || req.Password == "" {
			writeBadRequest(w, "Name, email, and password are required.")
			return
		}

		hash, err := utils.HashPassword(req.Password)
		if err != nil {
			log.Printf("Password hash error: %v", err)
			writeServerError(w, "Failed to process password. Please try again.")
			return
		}

		user := models.User{
			Email:    req.Email,
			Name:     req.Name,
			Password: hash,
			Provider: "local",
			Role:     "user",
		}

		userID, err := utils.UpsertUserInHasura(cfg, user)
		if err != nil {
			errStr := err.Error()
			if strings.Contains(errStr, "duplicate") || strings.Contains(errStr, "already exists") {
				writeConflict(w, "A user with this email already exists.")
				return
			}
			log.Printf("User creation error: %v", err)
			writeServerError(w, "Failed to create account. Please try again.")
			return
		}

		if err := utils.CreateEmptyUserProfile(cfg, userID); err != nil {
			log.Printf("Profile creation error: %v", err)
			writeServerError(w, "Account created but profile setup failed. Please contact support.")
			return
		}

		verifyURL := fmt.Sprintf("%s/auth/email/verify?token=demo", cfg.PublicBaseURL)
		log.Printf("Demo verification URL: %s", verifyURL)

		writeSuccess(w, map[string]interface{}{
			"message": "Account created successfully.",
			"user_id": userID,
		})
	}
}

func EmailVerify(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeSuccess(w, map[string]interface{}{
			"message": "Email verified (demo mode).",
		})
	}
}

func EmailLogin(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "Invalid request body. Please provide valid JSON.")
			return
		}

		req.Email = strings.TrimSpace(strings.ToLower(req.Email))

		if req.Email == "" || req.Password == "" {
			writeBadRequest(w, "Email and password are required.")
			return
		}

		user, err := utils.GetUserByEmail(cfg, req.Email)
		if err != nil {
			writeAuthError(w, "Invalid email or password.")
			return
		}

		if !utils.CheckPasswordHash(req.Password, user.Password) {
			writeAuthError(w, "Invalid email or password.")
			return
		}

		session, err := utils.GenerateJWT(cfg, user.ID, user.Role)
		if err != nil {
			log.Printf("JWT generation error: %v", err)
			writeServerError(w, "Failed to generate session. Please try again.")
			return
		}

		writeSuccess(w, map[string]interface{}{
			"access_token":  session.AccessToken,
			"refresh_token": session.RefreshToken,
			"expires_in":    session.ExpiresIn,
			"user": map[string]interface{}{
				"user_id":    user.ID,
				"email":      user.Email,
				"name":       user.Name,
				"avatar_url": nullIfEmpty(user.AvatarURL),
				"provider":   user.Provider,
				"role":       user.Role,
			},
		})
	}
}

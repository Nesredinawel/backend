package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"auth-service/models"
	"auth-service/utils"
)

// EmailSignup creates a new user with email & password
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

		// Hash password
		hash, err := utils.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}

		user := models.User{
			Email:    req.Email,
			Name:     req.Name,
			Password: hash,
			Role:     "user",
			Provider: "local",
		}

		userID, err := utils.UpsertUserInHasura(cfg, user)
		if err != nil {
			log.Printf("❌ upsert user failed: %v\n", err)
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}

		// Create empty profile
		if err := utils.CreateEmptyUserProfile(cfg, userID); err != nil {
			log.Printf("⚠️ failed to create user profile: %v\n", err)
		}

		// Generate session tokens
		session, err := utils.GenerateJWT(cfg, userID, user.Role)
		if err != nil {
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		resp := map[string]interface{}{
			"access_token":  session.AccessToken,
			"refresh_token": session.RefreshToken,
			"expires_in":    session.ExpiresIn,
			"user": map[string]string{
				"user_id":  userID,
				"email":    user.Email,
				"name":     user.Name,
				"provider": user.Provider,
				"role":     user.Role,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		pretty, _ := json.MarshalIndent(resp, "", "  ") // Pretty JSON
		w.Write(pretty)
	}
}

// EmailLogin verifies user credentials and returns JWT
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

		// Fetch user
		user, err := utils.GetUserByEmail(cfg, req.Email)
		if err != nil {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}

		if !utils.CheckPasswordHash(req.Password, user.Password) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		// Generate session tokens
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
		pretty, _ := json.MarshalIndent(resp, "", "  ")
		w.Write(pretty)
	}
}

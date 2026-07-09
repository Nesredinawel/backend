package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"auth-service/email"
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

		existing, svcErr := utils.GetUserByEmail(cfg, req.Email)
		if svcErr == nil && existing.ID != "" {
			writeConflict(w, "A user with this email already exists.")
			return
		}

		hash, pwdErr := utils.HashPassword(req.Password)
		if pwdErr != nil {
			log.Printf("Password hash error: %v", pwdErr)
			writeServerError(w, "Failed to process password. Please try again.")
			return
		}

		token, tokenErr := utils.GenerateVerificationToken()
		if tokenErr != nil {
			writeServerError(w, "Failed to generate verification token. Please try again.")
			return
		}

		pending := utils.PendingSignup{
			Name:         req.Name,
			Email:        req.Email,
			PasswordHash: hash,
		}

		if err := utils.SavePendingSignup(utils.Rdb, token, pending, 15*time.Minute); err != nil {
			log.Printf("Failed to save pending signup: %v", err)
			writeServerError(w, "Failed to process signup. Please try again.")
			return
		}

		verifyURL := cfg.PublicBaseURL + "/auth/email/verify?token=" + token
		provider := email.NewEmailProvider()
		if err := provider.SendVerificationEmail(req.Email, verifyURL); err != nil {
			log.Printf("Failed to send verification email: %v", err)
			writeServerError(w, "Account not created. Failed to send verification email. Please try again.")
			return
		}

		writeSuccess(w, map[string]interface{}{
			"message": "Verification email sent. Please check your inbox.",
			"email":   req.Email,
		})
	}
}

func EmailVerify(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			writeBadRequest(w, "Missing verification token.")
			return
		}

		pending, err := utils.GetPendingSignup(utils.Rdb, token)
		if err != nil {
			log.Printf("Failed to get pending signup: %v", err)
			writeBadRequest(w, "Invalid or expired verification token.")
			return
		}

		user := models.User{
			Email:    pending.Email,
			Name:     pending.Name,
			Password: pending.PasswordHash,
			Provider: "local",
			Role:     "user",
		}

		userID, svcErr := utils.UpsertUserInHasura(cfg, user)
		if svcErr != nil {
			log.Printf("User creation error on verify: %v", svcErr)
			writeServerError(w, "Failed to create account. Please try again.")
			return
		}

		if err := utils.CreateEmptyUserProfile(cfg, userID); err != nil {
			log.Printf("Profile creation error on verify: %v", err)
		}

		utils.DeletePendingSignup(utils.Rdb, token)

		writeSuccess(w, map[string]interface{}{
			"message": "Email verified successfully. You can now log in.",
			"user_id": userID,
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

		user, svcErr := utils.GetUserByEmail(cfg, req.Email)
		if svcErr != nil {
			writeAuthError(w, "Invalid email or password.")
			return
		}

		if !utils.CheckPasswordHash(req.Password, user.Password) {
			writeAuthError(w, "Invalid email or password.")
			return
		}

		session, jwtErr := utils.GenerateJWT(cfg, user.ID, user.Role)
		if jwtErr != nil {
			log.Printf("JWT generation error: %v", jwtErr)
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

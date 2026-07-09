package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"auth-service/email"
	"auth-service/middlewares"
	"auth-service/models"
	"auth-service/utils"
)

func validatePassword(password string) string {
	if len(password) < 8 {
		return "Password must be at least 8 characters."
	}
	return ""
}

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

		if msg := validatePassword(req.Password); msg != "" {
			writeBadRequest(w, msg)
			return
		}

		existing, svcErr := utils.GetUserByEmail(cfg, req.Email)
		if svcErr == nil && existing.ID != "" {
			writeSuccess(w, map[string]interface{}{
				"message": "If an account exists, a verification email has been sent.",
				"email":   req.Email,
			})
			return
		}

		hash, pwdErr := utils.HashPassword(req.Password)
		if pwdErr != nil {
			writeServerError(w, "Failed to process password. Please try again.")
			return
		}

		if utils.Rdb == nil {
			writeServerError(w, "Service temporarily unavailable. Please try again later.")
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
			writeServerError(w, "Failed to process signup. Please try again.")
			return
		}

		verifyURL := cfg.PublicBaseURL + "/auth/email/verify?token=" + token
		provider := email.NewEmailProvider()
		if err := provider.SendVerificationEmail(req.Email, verifyURL); err != nil {
			log.Printf("Failed to send verification email: %v", err)
			writeSuccess(w, map[string]interface{}{
				"message": "If an account exists, a verification email has been sent.",
				"email":   req.Email,
			})
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
			writeBadRequest(w, "Invalid or expired verification link.")
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

		if middlewares.CheckAccountLockout(req.Email) {
			writeAuthError(w, "Account temporarily locked due to too many failed attempts. Please try again later.")
			return
		}

		user, svcErr := utils.GetUserByEmail(cfg, req.Email)
		if svcErr != nil {
			middlewares.RecordFailedLogin(req.Email)
			writeAuthError(w, "Invalid email or password.")
			return
		}

		if !utils.CheckPasswordHash(req.Password, user.Password) {
			middlewares.RecordFailedLogin(req.Email)
			writeAuthError(w, "Invalid email or password.")
			return
		}

		middlewares.ResetFailedLogins(req.Email)

		session, jwtErr := utils.GenerateJWT(cfg, user.ID, user.Role)
		if jwtErr != nil {
			writeServerError(w, "Failed to generate session. Please try again.")
			return
		}

		storeRefreshToken(user.ID, session.RefreshToken)

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

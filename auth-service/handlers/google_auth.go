package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"auth-service/models"
	"auth-service/utils"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var oauthConfig *oauth2.Config

func InitGoogleOAuth(cfg utils.Config) {
	oauthConfig = &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
}

func GoogleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/auth/google/callback?code=demo_code_123", http.StatusTemporaryRedirect)
	}
}

func GoogleCallback(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := fmt.Sprintf("google.demo.%d@example.com", time.Now().Unix())
		name := "Demo Google User"

		user := models.User{
			Email:      email,
			Name:       name,
			AvatarURL:  "https://api.dicebear.com/7.x/avataaars/svg?seed=demo",
			Provider:   "google",
			ProviderID: "demo_google_id",
			Role:       "user",
		}

		userID, err := utils.UpsertUserInHasura(cfg, user)
		if err != nil {
			log.Printf("Google user upsert error: %v", err)
			writeServerError(w, "Failed to authenticate with Google. Please try again.")
			return
		}

		if err := utils.CreateEmptyUserProfile(cfg, userID); err != nil {
			log.Printf("Profile creation error: %v", err)
			writeServerError(w, "Authentication succeeded but profile setup failed. Please contact support.")
			return
		}

		session, err := utils.GenerateJWT(cfg, userID, "user")
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
				"user_id":    userID,
				"email":      email,
				"name":       name,
				"avatar_url": nullIfEmpty(user.AvatarURL),
				"provider":   "google",
				"role":       "user",
			},
		})
	}
}

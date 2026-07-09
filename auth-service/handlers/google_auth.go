package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"auth-service/models"
	"auth-service/utils"

	"github.com/google/uuid"
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
		if oauthConfig == nil {
			writeServerError(w, "Google OAuth is not configured.")
			return
		}
		state := uuid.New().String()
		url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func GoogleCallback(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			writeBadRequest(w, "Missing authorization code.")
			return
		}

		token, err := oauthConfig.Exchange(context.Background(), code)
		if err != nil {
			log.Printf("Google token exchange error: %v", err)
			writeServerError(w, "Failed to authenticate with Google. Please try again.")
			return
		}

		client := oauthConfig.Client(context.Background(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			log.Printf("Google userinfo fetch error: %v", err)
			writeServerError(w, "Failed to fetch user info from Google.")
			return
		}
		defer resp.Body.Close()

		var googleUser struct {
			ID      string `json:"id"`
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&googleUser); err != nil {
			log.Printf("Google userinfo decode error: %v", err)
			writeServerError(w, "Failed to process Google user data.")
			return
		}

		if googleUser.Email == "" {
			writeServerError(w, "Google account has no email associated.")
			return
		}

		user := models.User{
			Email:      googleUser.Email,
			Name:       googleUser.Name,
			AvatarURL:  googleUser.Picture,
			Provider:   "google",
			ProviderID: googleUser.ID,
			Role:       "user",
		}

		userID, svcErr := utils.UpsertUserInHasura(cfg, user)
		if svcErr != nil {
			log.Printf("Google user upsert error: %v", svcErr)
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
				"email":      googleUser.Email,
				"name":       googleUser.Name,
				"avatar_url": nullIfEmpty(googleUser.Picture),
				"provider":   "google",
				"role":       "user",
			},
		})
	}
}

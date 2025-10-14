package handlers

import (
	"context"
	"encoding/json"


	"log"
	"net/http"

	"auth-service/models"
	"auth-service/utils"

	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2"
)

var oauthConfig *oauth2.Config // global

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
	log.Println("✅ Google OAuth initialized")
}

func GoogleLogin() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if oauthConfig == nil {
			http.Error(w, "oauthConfig not initialized", http.StatusInternalServerError)
			return
		}
		url := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

func GoogleCallback(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if oauthConfig == nil {
			http.Error(w, "oauthConfig not initialized", http.StatusInternalServerError)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code", http.StatusBadRequest)
			return
		}

		token, err := oauthConfig.Exchange(context.Background(), code)
		if err != nil {
			log.Printf("token exchange error: %v\n", err)
			http.Error(w, "token exchange failed", http.StatusInternalServerError)
			return
		}

		client := oauthConfig.Client(context.Background(), token)
		resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
		if err != nil {
			http.Error(w, "failed fetching userinfo", http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		var gu struct {
			Email   string `json:"email"`
			Name    string `json:"name"`
			Picture string `json:"picture"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
			http.Error(w, "failed decoding userinfo", http.StatusInternalServerError)
			return
		}

		user := models.User{
			Email:     gu.Email,
			Name:      gu.Name,
			AvatarURL: gu.Picture,
		}

		userID, err := utils.UpsertUserInHasura(cfg, user)
		if err != nil {
			log.Printf("upsert user failed: %v\n", err)
			http.Error(w, "failed to upsert user", http.StatusInternalServerError)
			return
		}

		jwtToken, err := utils.GenerateJWT(cfg, userID)
		if err != nil {
			log.Printf("generate jwt failed: %v\n", err)
			http.Error(w, "failed to generate JWT", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"token":   jwtToken,
			"user_id": userID,
		})
	}
}

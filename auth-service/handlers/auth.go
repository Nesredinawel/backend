package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"auth-service/models"
	"auth-service/utils"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var oauthConfig *oauth2.Config

// GoogleLogin redirects user to Google's consent page
func GoogleLogin(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// state can be used to validate requests; for simplicity we use a static state here.
		url := oauthConfig.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	}
}

// GoogleCallback receives Google response, upserts user and returns JWT for Hasura usage
func GoogleCallback(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if oauthConfig == nil {
			http.Error(w, "oauthConfig not initialized", http.StatusInternalServerError)
			return
		}

		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "missing code in callback", http.StatusBadRequest)
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

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			log.Printf("google userinfo error: %s\n", string(bodyBytes))
			http.Error(w, "failed fetching userinfo", http.StatusInternalServerError)
			return
		}

		var gu struct {
			ID      string `json:"id"`
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
			http.Error(w, "failed to create or update user", http.StatusInternalServerError)
			return
		}

		jwtToken, err := utils.GenerateJWT(cfg, userID)
		if err != nil {
			log.Printf("generate jwt failed: %v\n", err)
			http.Error(w, "failed to generate token", http.StatusInternalServerError)
			return
		}

		// Return JSON response with token and user id
		respBody := map[string]string{
			"token":   jwtToken,
			"user_id": userID,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(respBody)
	}
}

// (Optional) reusable helper to ensure oauthConfig is present
func ensureOAuth() error {
	if oauthConfig == nil {
		return errors.New("oauth config not initialized: call /auth/google/login first")
	}
	return nil
}

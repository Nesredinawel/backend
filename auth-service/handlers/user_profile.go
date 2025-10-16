package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"auth-service/models"
	"auth-service/utils"
)

// GetUserProfile retrieves the currently authenticated user's profile
func GetUserProfile(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user_id from JWT context
		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			http.Error(w, "unauthorized: missing user ID", http.StatusUnauthorized)
			return
		}

		// Fetch profile from Hasura
		profile, err := utils.GetUserProfileFromHasura(cfg, userID)
		if err != nil {
			log.Printf("❌ Error fetching profile for user %s: %v", userID, err)
			http.Error(w, "failed to fetch user profile: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// Pretty JSON output
		w.Header().Set("Content-Type", "application/json")
		pretty, err := json.MarshalIndent(profile, "", "  ")
		if err != nil {
			log.Printf("❌ JSON marshal error: %v", err)
			http.Error(w, "failed to encode profile", http.StatusInternalServerError)
			return
		}
		w.Write(pretty)
	}
}

// UpdateUserProfile updates the user's bio or custom avatar
func UpdateUserProfile(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			http.Error(w, "unauthorized: missing user ID", http.StatusUnauthorized)
			return
		}

		// Decode request payload
		var req struct {
			Bio             *string `json:"bio,omitempty"`
			CustomAvatarURL *string `json:"custom_avatar_url,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request payload", http.StatusBadRequest)
			return
		}

		// Prepare update: dereference pointers for logging/debugging
		log.Printf("📝 Update payload for user %s: {Bio: %v, CustomAvatarURL: %v}", userID,
			func() interface{} { if req.Bio != nil { return *req.Bio }; return nil }(),
			func() interface{} { if req.CustomAvatarURL != nil { return *req.CustomAvatarURL }; return nil }(),
		)

		update := models.UserProfile{
			UserID:          userID,
			Bio:             req.Bio,
			CustomAvatarURL: req.CustomAvatarURL,
		}

		profile, err := utils.UpdateUserProfileInHasura(cfg, update)
		if err != nil {
			log.Printf("❌ Failed to update profile for user %s: %v", userID, err)
			http.Error(w, "failed to update user profile: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		pretty, _ := json.MarshalIndent(profile, "", "  ")
		w.Write(pretty)
	}
}


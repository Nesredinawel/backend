package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"auth-service/models"
	"auth-service/utils"
)

func GetUserProfile(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			writeAuthError(w, "Unauthorized. Please log in again.")
			return
		}

		profile, err := utils.GetUserProfileFromHasura(cfg, userID)
		if err != nil {
			log.Printf("Profile fetch error for user %s: %v", userID, err)
			writeServerError(w, "Failed to fetch profile. Please try again.")
			return
		}

		writeSuccess(w, map[string]interface{}{
			"profile": profile,
		})
	}
}

func UpdateUserProfile(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			writeAuthError(w, "Unauthorized. Please log in again.")
			return
		}

		var input models.UserProfileInput
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeBadRequest(w, "Invalid request body. Please provide valid JSON.")
			return
		}

		profile, err := utils.UpdateUserProfileInHasura(cfg, userID, input)
		if err != nil {
			log.Printf("Profile update error for user %s: %v", userID, err)
			writeServerError(w, "Failed to update profile. Please try again.")
			return
		}

		writeSuccess(w, map[string]interface{}{
			"profile": profile,
			"message": "Profile updated successfully",
		})
	}
}

package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"auth-service/utils"
)

func ChangePassword(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := r.Context().Value("user_id").(string)
		if !ok || userID == "" {
			writeAuthError(w, "Unauthorized. Please log in again.")
			return
		}

		var req struct {
			CurrentPassword string `json:"current_password"`
			NewPassword     string `json:"new_password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "Invalid request body.")
			return
		}
		if req.CurrentPassword == "" || req.NewPassword == "" {
			writeBadRequest(w, "current_password and new_password are required.")
			return
		}
		if msg := validatePassword(req.NewPassword); msg != "" {
			writeBadRequest(w, msg)
			return
		}

		user, svcErr := utils.GlobalHasura.GetUserByID(userID)
		if svcErr != nil {
			writeServerError(w, "Failed to verify identity. Please try again.")
			return
		}

		if !utils.CheckPasswordHash(req.CurrentPassword, user.Password) {
			writeAuthError(w, "Current password is incorrect.")
			return
		}

		newHash, err := utils.HashPassword(req.NewPassword)
		if err != nil {
			log.Printf("Password hash error: %v", err)
			writeServerError(w, "Failed to process new password.")
			return
		}

		_, svcErr = utils.GlobalHasura.UpdatePassword(userID, newHash)
		if svcErr != nil {
			writeServerError(w, "Failed to update password. Please try again.")
			return
		}

		writeSuccess(w, map[string]interface{}{"message": "Password changed successfully."})
	}
}

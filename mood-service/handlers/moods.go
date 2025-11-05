package handlers

import (
	"encoding/json"
	"net/http"

	"mood-service/middlewares"
	"mood-service/models"
	"mood-service/utils"
)

// CreateMood adds or updates today's mood
func CreateMood(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middlewares.CtxUserID).(string)

		var req models.Mood
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		req.UserID = userID

		id, err := utils.InsertOrUpdateMood(cfg, req)
		if err != nil {
			http.Error(w, "failed to insert/update mood", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"mood_id": id,
			"message": "Mood saved successfully (updated if already exists for today)",
		})
	}
}

// GetMoods returns all moods for the authenticated user
func GetMoods(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middlewares.CtxUserID).(string)

		moods, err := utils.GetUserMoods(cfg, userID)
		if err != nil {
			http.Error(w, "failed to fetch moods", http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": true,
			"moods":   moods,
		})
	}
}

package handlers

import (
	"encoding/json"
	"net/http"

	"mood-service/middlewares"
	"mood-service/models"
	"mood-service/utils"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// CreateMood adds or updates today's mood
func CreateMood(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.Context().Value(middlewares.CtxUserID).(string)

		var req models.Mood
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			utils.WriteJSONError(w, utils.NewBadRequestError("Invalid request body. Please provide valid JSON."), http.StatusBadRequest)
			return
		}

		req.UserID = userID

		id, err := utils.InsertOrUpdateMood(cfg, req)
		if err != nil {
			switch err.Code {
			case utils.ErrCodeValidation:
				utils.WriteJSONError(w, err, http.StatusBadRequest)
			case utils.ErrCodeHasura:
				utils.WriteJSONError(w, err, http.StatusInternalServerError)
			default:
				utils.WriteJSONError(w, err, http.StatusInternalServerError)
			}
			return
		}

		utils.PublishNotification(utils.Rdb, "mood_events", utils.NotificationEvent{
			UserID:        userID,
			Title:         "Mood Saved",
			Message:       "Your daily mood has been recorded.",
			SourceService: "mood-service",
			Action:        "MOOD_CREATE_OR_UPDATE",
			Meta: map[string]interface{}{
				"mood_id":   id,
				"mood":      req.MoodScore,
				"note":      req.Note,
				"timestamp": req.CreatedAt,
			},
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
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
			switch err.Code {
			case utils.ErrCodeHasura:
				utils.WriteJSONError(w, err, http.StatusInternalServerError)
			default:
				utils.WriteJSONError(w, err, http.StatusInternalServerError)
			}
			return
		}

		utils.PublishNotification(utils.Rdb, "mood_events", utils.NotificationEvent{
			UserID:        userID,
			Title:         "Mood History Viewed",
			Message:       "You checked your mood history.",
			SourceService: "mood-service",
			Action:        "MOOD_HISTORY_FETCH",
			Meta: map[string]interface{}{
				"mood_count": len(moods),
			},
		})

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"success": true,
			"moods":   moods,
		})
	}
}

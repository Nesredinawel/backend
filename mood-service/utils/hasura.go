package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"mood-service/models"
)

// InsertOrUpdateMood inserts or updates the user's mood for today
func InsertOrUpdateMood(cfg Config, mood models.Mood) (string, error) {
	today := time.Now().Format("2006-01-02")

	query := `
	mutation UpsertMood(
		$user_id: uuid!,
		$mood: String!,
		$emoji: String,
		$note: String,
		$mood_score: Int,
		$mood_date: date!
	) {
		insert_mood_service_moods_one(
			object: {
				user_id: $user_id,
				mood: $mood,
				emoji: $emoji,
				note: $note,
				mood_score: $mood_score,
				mood_date: $mood_date
			},
			on_conflict: {
				constraint: moods_user_id_mood_date_key,
				update_columns: [mood, emoji, note, mood_score, updated_at]
			}
		) {
			id
			mood_date
		}
	}`

	variables := map[string]interface{}{
		"user_id":    mood.UserID,
		"mood":       mood.Mood,
		"emoji":      mood.Emoji,
		"note":       mood.Note,
		"mood_score": mood.MoodScore,
		"mood_date":  today,
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("❌ Error sending request to Hasura: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Hasura returned %d: %s", resp.StatusCode, string(b))
		return "", fmt.Errorf("hasura returned non-200: %d", resp.StatusCode)
	}

	var respData struct {
		Data struct {
			InsertMoodServiceMoodsOne struct {
				ID       string `json:"id"`
				MoodDate string `json:"mood_date"`
			} `json:"insert_mood_service_moods_one"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.Unmarshal(b, &respData); err != nil {
		log.Printf("❌ Error decoding Hasura response: %v\nBody: %s", err, string(b))
		return "", err
	}

	if len(respData.Errors) > 0 {
		log.Printf("❌ Hasura errors: %v", respData.Errors)
		return "", fmt.Errorf("hasura errors: %v", respData.Errors)
	}

	log.Printf("✅ Mood saved for user %s on %s (ID: %s)", mood.UserID, today, respData.Data.InsertMoodServiceMoodsOne.ID)
	return respData.Data.InsertMoodServiceMoodsOne.ID, nil
}

// GetUserMoods fetches all moods for a given user
func GetUserMoods(cfg Config, userID string) ([]models.Mood, error) {
	query := `
	query GetMoods($user_id: uuid!) {
	  mood_service_moods(where: {user_id: {_eq: $user_id}}) {
	    id
	    mood
	    emoji
	    note
	    mood_date
		mood_score
	    created_at
	    updated_at
	  }
	}`
	variables := map[string]interface{}{
		"user_id": userID,
	}
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			Moods []models.Mood `json:"mood_service_moods"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, err
	}

	if len(respData.Errors) > 0 {
		return nil, fmt.Errorf("hasura errors: %v", respData.Errors)
	}

	return respData.Data.Moods, nil
}

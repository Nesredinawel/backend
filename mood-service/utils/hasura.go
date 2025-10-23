package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"mood-service/models"
)

// InsertMood inserts a mood record into Hasura
func InsertMood(cfg Config, mood models.Mood) (string, error) {
	query := `
	mutation InsertMood($user_id: uuid!, $mood: String!, $emoji: String, $note: String, $mood_date: date) {
	  insert_mood_service_moods_one(object: {
	    user_id: $user_id,
	    mood: $mood,
	    emoji: $emoji,
	    note: $note,
	    mood_date: $mood_date
	  }) {
	    id
	  }
	}`
	variables := map[string]interface{}{
		"user_id":   mood.UserID,
		"mood":      mood.Mood,
		"emoji":     mood.Emoji,
		"note":      mood.Note,
		"mood_date": mood.MoodDate,
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
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("hasura returned non-200: %d, %s", resp.StatusCode, string(b))
	}

	var respData struct {
		Data struct {
			InsertMoodServiceMoodsOne struct {
				ID string `json:"id"`
			} `json:"insert_mood_service_moods_one"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}
	if len(respData.Errors) > 0 {
		return "", fmt.Errorf("hasura errors: %v", respData.Errors)
	}

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

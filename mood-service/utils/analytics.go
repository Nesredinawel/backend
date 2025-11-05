package utils

import (
	"bytes"
	"encoding/json"
	"fmt"

	"net/http"

	"mood-service/models"
)

// GetMoodsForPeriod fetches mood rows for a user between fromDate and toDate (inclusive).
// fromDate and toDate format: "YYYY-MM-DD"
func GetMoodsForPeriod(cfg Config, userID, fromDate, toDate string) ([]models.Mood, error) {
	query := `
    query GetMoodsForPeriod($user_id: uuid!, $from: date!, $to: date!) {
      mood_service_moods(where: {
        user_id: {_eq: $user_id},
        mood_date: {_gte: $from, _lte: $to}
      }, order_by: {mood_date: asc, created_at: asc}) {
        id
        user_id
        mood
        emoji
        note
        mood_date
        mood_score
        created_at
        updated_at
      }
    }`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"user_id": userID,
			"from":    fromDate,
			"to":      toDate,
		},
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

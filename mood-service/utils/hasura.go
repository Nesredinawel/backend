package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"mood-service/models"
)

// moodScoreMap defines allowed moods and their corresponding scores (0–10 scale)
var moodScoreMap = map[string]int{
	"angry":     2,
	"not good":  4,
	"not good!": 4,
	"mediocre":  6,
	"good":      8,
	"very good": 10,
}

// reverseMoodMap maps score → mood for validation or score-based inference
var reverseMoodMap = map[int]string{
	2:  "Angry",
	4:  "Not Good!",
	6:  "Mediocre",
	8:  "Good",
	10: "Very Good",
}

// getMoodScore normalizes a mood string and returns its score
func getMoodScore(mood string) (int, *ServiceError) {
	mood = strings.TrimSpace(strings.ToLower(mood))
	if score, ok := moodScoreMap[mood]; ok {
		return score, nil
	}
	err := NewValidationError(
		fmt.Sprintf("'%s' is not a valid mood", mood),
		fmt.Sprintf("Allowed moods: %s. Please choose one of the listed moods.", strings.Join(keys(moodScoreMap), ", ")),
	)
	log.Printf("⚠️ Invalid mood input: '%s' — allowed: %v", mood, keys(moodScoreMap))
	return 0, err
}

// getMoodFromScore returns the correct mood string for a given score
func getMoodFromScore(score int) (string, *ServiceError) {
	if mood, ok := reverseMoodMap[score]; ok {
		return mood, nil
	}
	err := NewValidationError(
		fmt.Sprintf("Mood score %d is not valid", score),
		"Valid mood scores are: 2 (Angry), 4 (Not Good!), 6 (Mediocre), 8 (Good), 10 (Very Good). Please provide a score from this range.",
	)
	log.Printf("⚠️ Invalid mood_score input: %d — allowed: %v", score, keysInt(reverseMoodMap))
	return "", err
}

// normalizeAndValidateMood normalizes the mood input (either mood string or mood_score)
// and returns the canonical mood text and score. Returns error if neither is provided.
func normalizeAndValidateMood(mood models.Mood) (moodText string, moodScore int, err *ServiceError) {
	switch {
	case mood.Mood != "":
		score, err := getMoodScore(mood.Mood)
		if err != nil {
			log.Printf("❌ Mood validation failed: %v | user_id=%s", err, mood.UserID)
			return "", 0, err
		}
		return strings.Title(strings.ToLower(mood.Mood)), score, nil

	case mood.MoodScore != nil:
		text, err := getMoodFromScore(*mood.MoodScore)
		if err != nil {
			log.Printf("❌ Mood score validation failed: %v | user_id=%s", err, mood.UserID)
			return "", 0, err
		}
		return text, *mood.MoodScore, nil

	default:
		err := NewValidationError(
			"Missing mood information",
			"Either a mood name (e.g. 'good', 'angry') or a mood score (2, 4, 6, 8, 10) must be provided.",
		)
		log.Printf("❌ Missing input: neither mood nor mood_score provided | user_id=%s", mood.UserID)
		return "", 0, err
	}
}

// InsertOrUpdateMood inserts or updates the user's mood for today.
// User only sends mood_score OR mood — backend ensures both are consistent.
func InsertOrUpdateMood(cfg Config, mood models.Mood) (string, *ServiceError) {
	today := time.Now().Format("2006-01-02")

	moodText, moodScore, err := normalizeAndValidateMood(mood)
	if err != nil {
		return "", err
	}

	mood.Mood = moodText
	mood.MoodScore = &moodScore

	// GraphQL mutation
	query := `
	mutation UpsertMood(
		$user_id: uuid!,
		$mood: String!,
		$emoji: String,
		$note: String,
		$mood_score: Int!,
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
				constraint: moods_user_id_mood_date_idx,
				update_columns: [mood, emoji, note, mood_score, updated_at]
			}
		) {
			id
			mood_date
			mood_score
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

	resp, httpErr := http.DefaultClient.Do(req)
	if httpErr != nil {
		log.Printf("❌ Error sending request to Hasura: %v | user_id=%s", httpErr, mood.UserID)
		return "", NewHasuraError(
			"Failed to save mood due to a database connection error",
			"The mood service could not reach the database. Please try again later.",
		)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Hasura returned %d for user %s — body: %s", resp.StatusCode, mood.UserID, string(b))
		return "", NewHasuraError(
			"Database request failed",
			fmt.Sprintf("The database returned an unexpected status (%d). Please try again.", resp.StatusCode),
		)
	}

	var insertResp struct {
		Data struct {
			InsertMoodServiceMoodsOne struct {
				ID        string `json:"id"`
				MoodDate  string `json:"mood_date"`
				MoodScore int    `json:"mood_score"`
			} `json:"insert_mood_service_moods_one"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.Unmarshal(b, &insertResp); err != nil {
		log.Printf("❌ JSON decode failed for Hasura response: %v\nBody: %s | user_id=%s", err, string(b), mood.UserID)
		return "", NewServerError("Failed to process the server response. Please try again.")
	}

	if len(insertResp.Errors) > 0 {
		errMsg := fmt.Sprintf("%v", insertResp.Errors)
		log.Printf("❌ Hasura GraphQL errors for user %s: %s", mood.UserID, errMsg)
		return "", NewHasuraError(
			"Could not save your mood entry",
			errMsg,
		)
	}

	log.Printf("✅ Mood '%s' (score: %d) saved for user %s on %s (ID: %s)",
		mood.Mood, moodScore, mood.UserID, today, insertResp.Data.InsertMoodServiceMoodsOne.ID)

	return insertResp.Data.InsertMoodServiceMoodsOne.ID, nil
}

// GetUserMoods fetches all moods for a given user
func GetUserMoods(cfg Config, userID string) ([]models.Mood, *ServiceError) {
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
		log.Printf("❌ Error sending GetUserMoods request: %v | user_id=%s", err, userID)
		return nil, NewHasuraError(
			"Failed to fetch moods due to a database connection error",
			"The mood service could not reach the database. Please try again later.",
		)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	var respData struct {
		Data struct {
			Moods []models.Mood `json:"mood_service_moods"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.Unmarshal(b, &respData); err != nil {
		log.Printf("❌ JSON decode failed in GetUserMoods: %v\nBody: %s | user_id=%s", err, string(b), userID)
		return nil, NewServerError("Failed to process moods data. Please try again.")
	}

	if len(respData.Errors) > 0 {
		log.Printf("❌ Hasura GraphQL errors in GetUserMoods for user %s: %v", userID, respData.Errors)
		return nil, NewHasuraError(
			"Could not retrieve your mood history",
			"The database encountered an error while fetching your moods. Please try again.",
		)
	}

	log.Printf("✅ Retrieved %d moods for user %s", len(respData.Data.Moods), userID)
	return respData.Data.Moods, nil
}

// UpdateMoodByID updates a specific mood entry by its ID.
// Only the owner of the mood can update it (enforced via user_id filter).
func UpdateMoodByID(cfg Config, id, userID string, mood models.Mood) (string, *ServiceError) {
	moodText, moodScore, err := normalizeAndValidateMood(mood)
	if err != nil {
		return "", err
	}

	moodDate := time.Now().Format("2006-01-02")
	if mood.MoodDate != nil && *mood.MoodDate != "" {
		moodDate = *mood.MoodDate
	}

	query := `
	mutation UpdateMoodByID(
		$id: uuid!,
		$user_id: uuid!,
		$mood: String!,
		$emoji: String,
		$note: String,
		$mood_score: Int!,
		$mood_date: date!
	) {
		update_mood_service_moods(
			where: {
				id: {_eq: $id},
				user_id: {_eq: $user_id}
			},
			_set: {
				mood: $mood,
				emoji: $emoji,
				note: $note,
				mood_score: $mood_score,
				mood_date: $mood_date,
				updated_at: now()
			}
		) {
			affected_rows
			returning {
				id
				mood_date
				mood_score
			}
		}
	}`

	variables := map[string]interface{}{
		"id":         id,
		"user_id":    userID,
		"mood":       moodText,
		"emoji":      mood.Emoji,
		"note":       mood.Note,
		"mood_score": moodScore,
		"mood_date":  moodDate,
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, httpErr := http.DefaultClient.Do(req)
	if httpErr != nil {
		log.Printf("❌ Error sending UpdateMoodByID request: %v | id=%s user_id=%s", httpErr, id, userID)
		return "", NewHasuraError(
			"Failed to update mood due to a database connection error",
			"The mood service could not reach the database. Please try again later.",
		)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("❌ Hasura returned %d for UpdateMoodByID — id=%s body: %s", resp.StatusCode, id, string(b))
		return "", NewHasuraError(
			"Database request failed",
			fmt.Sprintf("The database returned an unexpected status (%d). Please try again.", resp.StatusCode),
		)
	}

	var updateResp struct {
		Data struct {
			UpdateMoodServiceMoods struct {
				AffectedRows int `json:"affected_rows"`
				Returning    []struct {
					ID        string `json:"id"`
					MoodDate  string `json:"mood_date"`
					MoodScore int    `json:"mood_score"`
				} `json:"returning"`
			} `json:"update_mood_service_moods"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.Unmarshal(b, &updateResp); err != nil {
		log.Printf("❌ JSON decode failed for UpdateMoodByID response: %v\nBody: %s | id=%s", err, string(b), id)
		return "", NewServerError("Failed to process the server response. Please try again.")
	}

	if len(updateResp.Errors) > 0 {
		errMsg := fmt.Sprintf("%v", updateResp.Errors)
		log.Printf("❌ Hasura GraphQL errors in UpdateMoodByID for id=%s: %s", id, errMsg)
		return "", NewHasuraError(
			"Could not update your mood entry",
			errMsg,
		)
	}

	if updateResp.Data.UpdateMoodServiceMoods.AffectedRows == 0 {
		log.Printf("⚠️ UpdateMoodByID: no rows affected — id=%s user_id=%s (not found or not owned)", id, userID)
		return "", NewNotFoundError("Mood entry not found or you do not have permission to update it.")
	}

	updated := updateResp.Data.UpdateMoodServiceMoods.Returning[0]
	log.Printf("✅ Mood '%s' (score: %d) updated for user %s on %s (ID: %s)",
		moodText, moodScore, userID, updated.MoodDate, updated.ID)

	return updated.ID, nil
}

// helper: get map keys (string)
func keys(m map[string]int) []string {
	k := make([]string, 0, len(m))
	for key := range m {
		k = append(k, key)
	}
	return k
}

// helper: get map keys (int)
func keysInt(m map[int]string) []int {
	k := make([]int, 0, len(m))
	for key := range m {
		k = append(k, key)
	}
	return k
}

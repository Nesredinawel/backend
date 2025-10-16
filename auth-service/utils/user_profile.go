package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"auth-service/models"
)

// ===============================
// 📌 Create Empty User Profile
// Called after signup or Google login
// ===============================
func CreateEmptyUserProfile(cfg Config, userID string) error {
	query := `
	mutation InsertUserProfile($user_id: uuid!) {
		insert_user_profiles_one(object: {user_id: $user_id}) {
			id
			user_id
		}
	}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"user_id": userID,
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to create user profile: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to create user profile, status: %d", resp.StatusCode)
	}
	return nil
}

// ===============================
// 📌 Get User Profile
// ===============================
func GetUserProfileFromHasura(cfg Config, userID string) (*models.UserProfile, error) {
	query := `
	query GetUserProfile($user_id: uuid!) {
		user_profiles(where: {user_id: {_eq: $user_id}}) {
			id
			user_id
			bio
			custom_avatar_url
			created_at
			updated_at
		}
	}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"user_id": userID,
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch profile: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch profile, status: %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			UserProfiles []models.UserProfile `json:"user_profiles"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	if len(result.Data.UserProfiles) == 0 {
		return nil, fmt.Errorf("profile not found for user %s", userID)
	}

	return &result.Data.UserProfiles[0], nil
}

// ===============================
// 📌 Update User Profile (Upsert)
// ===============================
func UpdateUserProfileInHasura(cfg Config, profile models.UserProfile) (*models.UserProfile, error) {
	query := `
	mutation UpsertUserProfile($user_id: uuid!, $bio: String, $custom_avatar_url: String) {
		insert_user_profiles_one(
			object: { user_id: $user_id, bio: $bio, custom_avatar_url: $custom_avatar_url },
			on_conflict: {
				constraint: user_profiles_user_id_key,
				update_columns: [bio, custom_avatar_url, updated_at]
			}
		) {
			id
			user_id
			bio
			custom_avatar_url
			created_at
			updated_at
		}
	}`

	// Helper function to dereference or return nil
	nilIfEmpty := func(s *string) interface{} {
		if s == nil {
			return nil
		}
		return *s
	}

	// Build variables with proper dereferencing
	variables := map[string]interface{}{
		"user_id":           profile.UserID,
		"bio":               nilIfEmpty(profile.Bio),
		"custom_avatar_url": nilIfEmpty(profile.CustomAvatarURL),
	}

	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to update profile: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to update profile, status: %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			InsertUserProfilesOne models.UserProfile `json:"insert_user_profiles_one"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}

	return &result.Data.InsertUserProfilesOne, nil
}

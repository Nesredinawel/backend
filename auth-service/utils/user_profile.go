package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"auth-service/models"
)

func doHasuraRequest(cfg Config, query string, variables map[string]interface{}) (*http.Response, error) {
	payload := map[string]interface{}{"query": query, "variables": variables}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func decodeHasuraResponse(resp *http.Response, target interface{}) *ServiceError {
	defer resp.Body.Close()

	var wrapper struct {
		Data   interface{}   `json:"data"`
		Errors []interface{} `json:"errors"`
	}
	wrapper.Data = target

	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return NewServerError(fmt.Sprintf("Failed to decode response: %v", err))
	}
	if len(wrapper.Errors) > 0 {
		return NewHasuraError("Database query failed.", fmt.Sprintf("%v", wrapper.Errors))
	}
	return nil
}

func CreateEmptyUserProfile(cfg Config, userID string) *ServiceError {
	now := time.Now().UTC().Format(time.RFC3339)

	resp, err := doHasuraRequest(cfg, `
		mutation UpdateProfile($user_id: uuid!, $updated_at: timestamptz) {
			update_auth_service_user_profiles(
				where: {user_id: {_eq: $user_id}},
				_set: {updated_at: $updated_at}
			) { affected_rows }
		}`,
		map[string]interface{}{"user_id": userID, "updated_at": now},
	)
	if err != nil {
		return NewHasuraError("Database connection error.", "Please try again later.")
	}

	var updateResult struct {
		UpdateAuthServiceUserProfiles struct {
			AffectedRows int `json:"affected_rows"`
		} `json:"update_auth_service_user_profiles"`
	}
	if svcErr := decodeHasuraResponse(resp, &updateResult); svcErr != nil {
		return svcErr
	}

	if updateResult.UpdateAuthServiceUserProfiles.AffectedRows > 0 {
		return nil
	}

	resp, err = doHasuraRequest(cfg, `
		mutation InsertProfile($user_id: uuid!, $updated_at: timestamptz) {
			insert_auth_service_user_profiles_one(object: {
				user_id: $user_id,
				updated_at: $updated_at
			}) { id user_id }
		}`,
		map[string]interface{}{"user_id": userID, "updated_at": now},
	)
	if err != nil {
		return NewHasuraError("Database connection error.", "Please try again later.")
	}

	var insertResult struct {
		InsertAuthServiceUserProfilesOne *struct {
			ID     string `json:"id"`
			UserID string `json:"user_id"`
		} `json:"insert_auth_service_user_profiles_one"`
	}
	if svcErr := decodeHasuraResponse(resp, &insertResult); svcErr != nil {
		return svcErr
	}
	if insertResult.InsertAuthServiceUserProfilesOne == nil {
		return NewServerError("Failed to create user profile.")
	}

	return nil
}

func GetUserProfileFromHasura(cfg Config, userID string) (*models.UserProfile, *ServiceError) {
	resp, err := doHasuraRequest(cfg, `
		query GetUserProfile($user_id: uuid!) {
			user: auth_service_users_by_pk(id: $user_id) {
				id name email avatar_url
			}
			profile: auth_service_user_profiles(where: {user_id: {_eq: $user_id}}) {
				id user_id bio custom_avatar_url created_at updated_at
			}
		}`,
		map[string]interface{}{"user_id": userID},
	)
	if err != nil {
		return nil, NewHasuraError("Database connection error.", "Please try again later.")
	}

	var rawResult struct {
		User *struct {
			ID        string  `json:"id"`
			Name      string  `json:"name"`
			Email     string  `json:"email"`
			AvatarURL *string `json:"avatar_url"`
		} `json:"user"`
		Profile []struct {
			ID              string  `json:"id"`
			UserID          string  `json:"user_id"`
			Bio             *string `json:"bio"`
			CustomAvatarURL *string `json:"custom_avatar_url"`
			CreatedAt       *string `json:"created_at"`
			UpdatedAt       *string `json:"updated_at"`
		} `json:"profile"`
	}
	if svcErr := decodeHasuraResponse(resp, &rawResult); svcErr != nil {
		return nil, svcErr
	}

	if rawResult.User == nil {
		log.Printf("User not found: %s", userID)
		return nil, NewNotFoundError("User not found.")
	}

	profile := &models.UserProfile{
		UserID:    rawResult.User.ID,
		Name:      rawResult.User.Name,
		Email:     rawResult.User.Email,
		AvatarURL: rawResult.User.AvatarURL,
	}

	if len(rawResult.Profile) > 0 {
		p := rawResult.Profile[0]
		profile.ID = p.ID
		profile.Bio = p.Bio
		profile.CustomAvatarURL = p.CustomAvatarURL
		profile.CreatedAt = p.CreatedAt
		profile.UpdatedAt = p.UpdatedAt
	}

	return profile, nil
}

func UpdateUserProfileInHasura(cfg Config, userID string, input models.UserProfileInput) (*models.UserProfile, *ServiceError) {
	now := time.Now().UTC().Format(time.RFC3339)

	resp, err := doHasuraRequest(cfg, `
		mutation UpdateProfile(
			$user_id: uuid!,
			$bio: String,
			$custom_avatar_url: String,
			$updated_at: timestamptz
		) {
			update_auth_service_user_profiles(
				where: {user_id: {_eq: $user_id}},
				_set: {bio: $bio, custom_avatar_url: $custom_avatar_url, updated_at: $updated_at}
			) { affected_rows }
		}`,
		map[string]interface{}{
			"user_id":           userID,
			"bio":               input.Bio,
			"custom_avatar_url": input.CustomAvatarURL,
			"updated_at":        now,
		},
	)
	if err != nil {
		return nil, NewHasuraError("Database connection error.", "Please try again later.")
	}

	var updateResult struct {
		UpdateAuthServiceUserProfiles struct {
			AffectedRows int `json:"affected_rows"`
		} `json:"update_auth_service_user_profiles"`
	}
	if svcErr := decodeHasuraResponse(resp, &updateResult); svcErr != nil {
		return nil, svcErr
	}

	if updateResult.UpdateAuthServiceUserProfiles.AffectedRows == 0 {
		resp, err = doHasuraRequest(cfg, `
			mutation InsertProfile(
				$user_id: uuid!,
				$bio: String,
				$custom_avatar_url: String,
				$updated_at: timestamptz
			) {
				insert_auth_service_user_profiles_one(object: {
					user_id: $user_id,
					bio: $bio,
					custom_avatar_url: $custom_avatar_url,
					updated_at: $updated_at
				}) {
					id user_id bio custom_avatar_url created_at updated_at
				}
			}`,
			map[string]interface{}{
				"user_id":           userID,
				"bio":               input.Bio,
				"custom_avatar_url": input.CustomAvatarURL,
				"updated_at":        now,
			},
		)
		if err != nil {
			return nil, NewHasuraError("Database connection error.", "Please try again later.")
		}

		var insertResult struct {
			InsertAuthServiceUserProfilesOne *struct {
				ID              string  `json:"id"`
				UserID          string  `json:"user_id"`
				Bio             *string `json:"bio"`
				CustomAvatarURL *string `json:"custom_avatar_url"`
				CreatedAt       *string `json:"created_at"`
				UpdatedAt       *string `json:"updated_at"`
			} `json:"insert_auth_service_user_profiles_one"`
		}
		if svcErr := decodeHasuraResponse(resp, &insertResult); svcErr != nil {
			return nil, svcErr
		}
		if insertResult.InsertAuthServiceUserProfilesOne == nil {
			return nil, NewServerError("Failed to create profile record.")
		}
	}

	return GetUserProfileFromHasura(cfg, userID)
}

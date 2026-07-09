package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"auth-service/models"
)

type HasuraClient struct {
	Endpoint string
	Secret   string
	client   *http.Client
}

func NewHasuraClient(cfg Config) *HasuraClient {
	return &HasuraClient{
		Endpoint: cfg.HasuraEndpoint,
		Secret:   cfg.HasuraAdminSecret,
		client:   HTTPClient,
	}
}

func (hc *HasuraClient) Do(query string, variables map[string]interface{}, target interface{}) *ServiceError {
	payload := map[string]interface{}{"query": query, "variables": variables}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", hc.Endpoint, bytes.NewBuffer(body))
	if err != nil {
		return NewServerError("Failed to create request.")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", hc.Secret)

	resp, httpErr := hc.client.Do(req)
	if httpErr != nil {
		return NewHasuraError("Database connection error.", "Please try again later.")
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return NewHasuraError("Database request failed.", fmt.Sprintf("Status %d.", resp.StatusCode))
	}

	var wrapper struct {
		Data   interface{}   `json:"data"`
		Errors []interface{} `json:"errors"`
	}
	wrapper.Data = target

	if err := json.Unmarshal(respBody, &wrapper); err != nil {
		return NewServerError(fmt.Sprintf("Failed to decode response: %v", err))
	}
	if len(wrapper.Errors) > 0 {
		return NewHasuraError("Database operation failed.", fmt.Sprintf("%v", wrapper.Errors))
	}
	return nil
}

func (hc *HasuraClient) UpsertUser(user models.User) (string, *ServiceError) {
	existing, err := hc.GetUserByEmail(user.Email)
	if err == nil && existing.ID != "" {
		if existing.Provider != user.Provider {
			if user.Provider == "local" && existing.Password == "" && user.Password != "" {
				return hc.updateUserPasswordAndProvider(existing.ID, user.Password, user.Provider)
			}
			if user.Provider != "local" && existing.Provider == "local" && existing.ProviderID == "" {
				return hc.updateUserProvider(existing.ID, user.Provider, user.ProviderID)
			}
			return hc.updateUserProvider(existing.ID, user.Provider, user.ProviderID)
		}
		if existing.Provider == "local" && existing.Password == "" && user.Password != "" {
			return hc.updateUserPassword(existing.ID, user.Password)
		}
		return existing.ID, nil
	}

	var result struct {
		Data struct {
			InsertAuthServiceUsersOne struct {
				ID string `json:"id"`
			} `json:"insert_auth_service_users_one"`
		} `json:"data"`
	}
	query := `mutation InsertUser($email: String!, $name: String, $avatar_url: String, $password: String, $provider: String, $provider_id: String, $role: String) {
	  insert_auth_service_users_one(object: {email: $email, name: $name, avatar_url: $avatar_url, password: $password, provider: $provider, provider_id: $provider_id, role: $role}) { id }
	}`
	variables := map[string]interface{}{
		"email": user.Email, "name": user.Name, "avatar_url": user.AvatarURL,
		"password": user.Password, "provider": user.Provider,
		"provider_id": user.ProviderID, "role": user.Role,
	}
	if svcErr := hc.Do(query, variables, &result.Data); svcErr != nil {
		log.Printf("InsertUser error: %v | email=%s", svcErr, user.Email)
		return "", svcErr
	}
	id := result.Data.InsertAuthServiceUsersOne.ID
	if id == "" {
		return "", NewServerError("No user ID returned.")
	}
	log.Printf("User created: %s (%s)", user.Email, id)
	return id, nil
}

func (hc *HasuraClient) GetUserByID(userID string) (models.User, *ServiceError) {
	var result struct {
		Data struct {
			AuthServiceUsersByPk *models.User `json:"auth_service_users_by_pk"`
		} `json:"data"`
	}
	query := `query GetUserByID($id: uuid!) { auth_service_users_by_pk(id: $id) { id email name password avatar_url provider provider_id role } }`
	if svcErr := hc.Do(query, map[string]interface{}{"id": userID}, &result.Data); svcErr != nil {
		return models.User{}, svcErr
	}
	if result.Data.AuthServiceUsersByPk == nil {
		return models.User{}, NewNotFoundError("User not found.")
	}
	return *result.Data.AuthServiceUsersByPk, nil
}

func (hc *HasuraClient) GetUserByEmail(email string) (models.User, *ServiceError) {
	var result struct {
		Data struct {
			AuthServiceUsers []models.User `json:"auth_service_users"`
		} `json:"data"`
	}
	query := `query GetUser($email: String!) { auth_service_users(where: {email: {_eq: $email}}) { id email name password avatar_url provider provider_id role } }`
	if svcErr := hc.Do(query, map[string]interface{}{"email": email}, &result.Data); svcErr != nil {
		log.Printf("GetUserByEmail error: %v | email=%s", svcErr, email)
		return models.User{}, svcErr
	}
	if len(result.Data.AuthServiceUsers) == 0 {
		return models.User{}, NewNotFoundError("User not found.")
	}
	return result.Data.AuthServiceUsers[0], nil
}

func (hc *HasuraClient) updateUserProvider(userID, provider, providerID string) (string, *ServiceError) {
	var result struct {
		Data struct {
			UpdateAuthServiceUsersByPk struct {
				ID string `json:"id"`
			} `json:"update_auth_service_users_by_pk"`
		} `json:"data"`
	}
	query := `mutation UpdateProvider($id: uuid!, $provider: String!, $provider_id: String) { update_auth_service_users_by_pk(pk_columns: {id: $id}, _set: {provider: $provider, provider_id: $provider_id}) { id } }`
	if svcErr := hc.Do(query, map[string]interface{}{"id": userID, "provider": provider, "provider_id": providerID}, &result.Data); svcErr != nil {
		return "", svcErr
	}
	return result.Data.UpdateAuthServiceUsersByPk.ID, nil
}

func (hc *HasuraClient) UpdatePassword(userID, password string) (string, *ServiceError) {
	return hc.updateUserPassword(userID, password)
}

func (hc *HasuraClient) updateUserPassword(userID, password string) (string, *ServiceError) {
	var result struct {
		Data struct {
			UpdateAuthServiceUsersByPk struct {
				ID string `json:"id"`
			} `json:"update_auth_service_users_by_pk"`
		} `json:"data"`
	}
	query := `mutation UpdatePassword($id: uuid!, $password: String!) { update_auth_service_users_by_pk(pk_columns: {id: $id}, _set: {password: $password}) { id } }`
	if svcErr := hc.Do(query, map[string]interface{}{"id": userID, "password": password}, &result.Data); svcErr != nil {
		return "", svcErr
	}
	return result.Data.UpdateAuthServiceUsersByPk.ID, nil
}

func (hc *HasuraClient) updateUserPasswordAndProvider(userID, password, provider string) (string, *ServiceError) {
	var result struct {
		Data struct {
			UpdateAuthServiceUsersByPk struct {
				ID string `json:"id"`
			} `json:"update_auth_service_users_by_pk"`
		} `json:"data"`
	}
	query := `mutation UpdatePasswordAndProvider($id: uuid!, $password: String, $provider: String) { update_auth_service_users_by_pk(pk_columns: {id: $id}, _set: {password: $password, provider: $provider}) { id } }`
	if svcErr := hc.Do(query, map[string]interface{}{"id": userID, "password": password, "provider": provider}, &result.Data); svcErr != nil {
		return "", svcErr
	}
	return result.Data.UpdateAuthServiceUsersByPk.ID, nil
}

func (hc *HasuraClient) CreateEmptyUserProfile(userID string) *ServiceError {
	now := time.Now().UTC().Format(time.RFC3339)

	var updateResult struct {
		Data struct {
			UpdateAuthServiceUserProfiles struct {
				AffectedRows int `json:"affected_rows"`
			} `json:"update_auth_service_user_profiles"`
		} `json:"data"`
	}
	if svcErr := hc.Do(`mutation UpdateProfile($user_id: uuid!, $updated_at: timestamptz) { update_auth_service_user_profiles(where: {user_id: {_eq: $user_id}}, _set: {updated_at: $updated_at}) { affected_rows } }`,
		map[string]interface{}{"user_id": userID, "updated_at": now}, &updateResult.Data); svcErr != nil {
		return svcErr
	}
	if updateResult.Data.UpdateAuthServiceUserProfiles.AffectedRows > 0 {
		return nil
	}

	var insertResult struct {
		Data struct {
			InsertAuthServiceUserProfilesOne *struct {
				ID     string `json:"id"`
				UserID string `json:"user_id"`
			} `json:"insert_auth_service_user_profiles_one"`
		} `json:"data"`
	}
	if svcErr := hc.Do(`mutation InsertProfile($user_id: uuid!, $updated_at: timestamptz) { insert_auth_service_user_profiles_one(object: {user_id: $user_id, updated_at: $updated_at}) { id user_id } }`,
		map[string]interface{}{"user_id": userID, "updated_at": now}, &insertResult.Data); svcErr != nil {
		return svcErr
	}
	if insertResult.Data.InsertAuthServiceUserProfilesOne == nil {
		return NewServerError("Failed to create user profile.")
	}
	return nil
}

func (hc *HasuraClient) GetUserProfile(userID string) (*models.UserProfile, *ServiceError) {
	var rawResult struct {
		Data struct {
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
		} `json:"data"`
	}
	query := `query GetUserProfile($user_id: uuid!) { user: auth_service_users_by_pk(id: $user_id) { id name email avatar_url } profile: auth_service_user_profiles(where: {user_id: {_eq: $user_id}}) { id user_id bio custom_avatar_url created_at updated_at } }`
	if svcErr := hc.Do(query, map[string]interface{}{"user_id": userID}, &rawResult.Data); svcErr != nil {
		return nil, svcErr
	}
	if rawResult.Data.User == nil {
		return nil, NewNotFoundError("User not found.")
	}
	profile := &models.UserProfile{
		UserID:    rawResult.Data.User.ID,
		Name:      rawResult.Data.User.Name,
		Email:     rawResult.Data.User.Email,
		AvatarURL: rawResult.Data.User.AvatarURL,
	}
	if len(rawResult.Data.Profile) > 0 {
		p := rawResult.Data.Profile[0]
		profile.ID = p.ID
		profile.Bio = p.Bio
		profile.CustomAvatarURL = p.CustomAvatarURL
		profile.CreatedAt = p.CreatedAt
		profile.UpdatedAt = p.UpdatedAt
	}
	return profile, nil
}

func (hc *HasuraClient) UpdateUserProfile(userID string, input models.UserProfileInput) (*models.UserProfile, *ServiceError) {
	now := time.Now().UTC().Format(time.RFC3339)

	var updateResult struct {
		Data struct {
			UpdateAuthServiceUserProfiles struct {
				AffectedRows int `json:"affected_rows"`
			} `json:"update_auth_service_user_profiles"`
		} `json:"data"`
	}
	if svcErr := hc.Do(`mutation UpdateProfile($user_id: uuid!, $bio: String, $custom_avatar_url: String, $updated_at: timestamptz) { update_auth_service_user_profiles(where: {user_id: {_eq: $user_id}}, _set: {bio: $bio, custom_avatar_url: $custom_avatar_url, updated_at: $updated_at}) { affected_rows } }`,
		map[string]interface{}{"user_id": userID, "bio": input.Bio, "custom_avatar_url": input.CustomAvatarURL, "updated_at": now}, &updateResult.Data); svcErr != nil {
		return nil, svcErr
	}

	if updateResult.Data.UpdateAuthServiceUserProfiles.AffectedRows == 0 {
		var insertResult struct {
			Data struct {
				InsertAuthServiceUserProfilesOne *struct {
					ID              string  `json:"id"`
					UserID          string  `json:"user_id"`
					Bio             *string `json:"bio"`
					CustomAvatarURL *string `json:"custom_avatar_url"`
					CreatedAt       *string `json:"created_at"`
					UpdatedAt       *string `json:"updated_at"`
				} `json:"insert_auth_service_user_profiles_one"`
			} `json:"data"`
		}
		if svcErr := hc.Do(`mutation InsertProfile($user_id: uuid!, $bio: String, $custom_avatar_url: String, $updated_at: timestamptz) { insert_auth_service_user_profiles_one(object: {user_id: $user_id, bio: $bio, custom_avatar_url: $custom_avatar_url, updated_at: $updated_at}) { id user_id bio custom_avatar_url created_at updated_at } }`,
			map[string]interface{}{"user_id": userID, "bio": input.Bio, "custom_avatar_url": input.CustomAvatarURL, "updated_at": now}, &insertResult.Data); svcErr != nil {
			return nil, svcErr
		}
		if insertResult.Data.InsertAuthServiceUserProfilesOne == nil {
			return nil, NewServerError("Failed to create profile record.")
		}
	}

	return hc.GetUserProfile(userID)
}

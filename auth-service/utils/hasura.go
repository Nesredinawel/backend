package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"auth-service/models"
)

// UpsertUserInHasura inserts or updates a user record in Hasura (auth_service.users)
func UpsertUserInHasura(cfg Config, user models.User) (string, error) {
	existing, err := GetUserByEmail(cfg, user.Email)
	if err == nil && existing.ID != "" {
		// 🧠 Existing user found → handle linking or updating provider/password
		fmt.Printf("[INFO] Existing user found for %s (provider=%s)\n", existing.Email, existing.Provider)

		// 1️⃣ If provider differs (e.g., Google → local or vice versa)
		if existing.Provider != user.Provider {
			fmt.Printf("[INFO] Linking new provider '%s' for user %s\n", user.Provider, existing.Email)

			// If user signs up with Email after Google login — add password
			if user.Provider == "local" && existing.Password == "" && user.Password != "" {
				fmt.Println("[INFO] Adding password to Google-linked account.")
				return UpdateUserPasswordAndProvider(cfg, existing.ID, user.Password, user.Provider)
			}

			// If user signs up with Google after Email login — just link provider_id
			if user.Provider != "local" && existing.Provider == "local" && existing.ProviderID == "" {
				fmt.Println("[INFO] Adding Google provider_id to existing local account.")
				return UpdateUserProvider(cfg, existing.ID, user.Provider, user.ProviderID)
			}

			// Default case: just ensure provider consistency
			return UpdateUserProvider(cfg, existing.ID, user.Provider, user.ProviderID)
		}

		// 2️⃣ If same provider but missing password (e.g., user sets password later)
		if existing.Provider == "local" && existing.Password == "" && user.Password != "" {
			fmt.Println("[INFO] Setting password for existing local user without password.")
			return UpdateUserPassword(cfg, existing.ID, user.Password)
		}

		// ✅ Return existing user if everything already matches
		return existing.ID, nil
	}

	// 🆕 Create new user if not exists
	query := `
	mutation InsertUser(
		$email: String!, 
		$name: String, 
		$avatar_url: String, 
		$password: String, 
		$provider: String, 
		$provider_id: String, 
		$role: String
	) {
	  insert_auth_service_users_one(
	    object: {
	      email: $email,
	      name: $name,
	      avatar_url: $avatar_url,
	      password: $password,
	      provider: $provider,
	      provider_id: $provider_id,
	      role: $role
	    }
	  ) { id }
	}`

	variables := map[string]interface{}{
		"email":       user.Email,
		"name":        user.Name,
		"avatar_url":  user.AvatarURL,
		"password":    user.Password,
		"provider":    user.Provider,
		"provider_id": user.ProviderID,
		"role":        user.Role,
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
		return "", fmt.Errorf("hasura returned non-200: %d, body: %s", resp.StatusCode, string(b))
	}

	var respData struct {
		Data struct {
			InsertAuthServiceUsersOne struct {
				ID string `json:"id"`
			} `json:"insert_auth_service_users_one"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}

	if respData.Data.InsertAuthServiceUsersOne.ID == "" {
		return "", errors.New("no user id returned from hasura")
	}

	return respData.Data.InsertAuthServiceUsersOne.ID, nil
}

// UpdateUserProvider links a new OAuth provider to an existing user
func UpdateUserProvider(cfg Config, userID, provider, providerID string) (string, error) {
	query := `
	mutation UpdateProvider($id: uuid!, $provider: String!, $provider_id: String) {
	  update_auth_service_users_by_pk(
	    pk_columns: {id: $id},
	    _set: {provider: $provider, provider_id: $provider_id}
	  ) { id }
	}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"id":          userID,
			"provider":    provider,
			"provider_id": providerID,
		},
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
		return "", fmt.Errorf("hasura update error: %s", string(b))
	}

	var respData struct {
		Data struct {
			UpdateAuthServiceUsersByPk struct {
				ID string `json:"id"`
			} `json:"update_auth_service_users_by_pk"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}

	return respData.Data.UpdateAuthServiceUsersByPk.ID, nil
}

// UpdateUserPassword sets a new password for an existing user
func UpdateUserPassword(cfg Config, userID, password string) (string, error) {
	query := `
	mutation UpdatePassword($id: uuid!, $password: String!) {
	  update_auth_service_users_by_pk(
	    pk_columns: {id: $id},
	    _set: {password: $password}
	  ) { id }
	}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"id":       userID,
			"password": password,
		},
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

	var respData struct {
		Data struct {
			UpdateAuthServiceUsersByPk struct {
				ID string `json:"id"`
			} `json:"update_auth_service_users_by_pk"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}

	return respData.Data.UpdateAuthServiceUsersByPk.ID, nil
}

// UpdateUserPasswordAndProvider — when user first had Google, then sets password
func UpdateUserPasswordAndProvider(cfg Config, userID, password, provider string) (string, error) {
	query := `
	mutation UpdatePasswordAndProvider($id: uuid!, $password: String, $provider: String) {
	  update_auth_service_users_by_pk(
	    pk_columns: {id: $id},
	    _set: {password: $password, provider: $provider}
	  ) { id }
	}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"id":       userID,
			"password": password,
			"provider": provider,
		},
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

	var respData struct {
		Data struct {
			UpdateAuthServiceUsersByPk struct {
				ID string `json:"id"`
			} `json:"update_auth_service_users_by_pk"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}

	return respData.Data.UpdateAuthServiceUsersByPk.ID, nil
}

// GetUserByEmail fetches a user from Hasura by email (auth_service.users)
func GetUserByEmail(cfg Config, email string) (models.User, error) {
	query := `
	query GetUser($email: String!) {
	  auth_service_users(where: {email: {_eq: $email}}) {
	    id
	    email
	    name
	    password
	    avatar_url
	    provider
	    provider_id
	    role
	  }
	}`

	payload := map[string]interface{}{
		"query":     query,
		"variables": map[string]interface{}{"email": email},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	fmt.Println("[DEBUG] → Fetching user by email:", email)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return models.User{}, err
	}
	defer resp.Body.Close()

	var respData struct {
		Data struct {
			AuthServiceUsers []models.User `json:"auth_service_users"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return models.User{}, err
	}
	if len(respData.Errors) > 0 {
		fmt.Println("[DEBUG] Hasura error:", respData.Errors)
	}
	if len(respData.Data.AuthServiceUsers) == 0 {
		return models.User{}, errors.New("user not found")
	}

	fmt.Println("[DEBUG] ⇦ Found user:", respData.Data.AuthServiceUsers[0].Email)
	return respData.Data.AuthServiceUsers[0], nil
}

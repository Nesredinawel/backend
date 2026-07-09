package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"auth-service/models"
)

// UpsertUserInHasura inserts or updates a user record in Hasura (auth_service.users)
func UpsertUserInHasura(cfg Config, user models.User) (string, *ServiceError) {
	existing, err := GetUserByEmail(cfg, user.Email)
	if err == nil && existing.ID != "" {
		if existing.Provider != user.Provider {
			if user.Provider == "local" && existing.Password == "" && user.Password != "" {
				return UpdateUserPasswordAndProvider(cfg, existing.ID, user.Password, user.Provider)
			}
			if user.Provider != "local" && existing.Provider == "local" && existing.ProviderID == "" {
				return UpdateUserProvider(cfg, existing.ID, user.Provider, user.ProviderID)
			}
			return UpdateUserProvider(cfg, existing.ID, user.Provider, user.ProviderID)
		}

		if existing.Provider == "local" && existing.Password == "" && user.Password != "" {
			return UpdateUserPassword(cfg, existing.ID, user.Password)
		}

		return existing.ID, nil
	}

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

	payload := map[string]interface{}{"query": query, "variables": variables}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, httpErr := http.DefaultClient.Do(req)
	if httpErr != nil {
		log.Printf("Hasura InsertUser request error: %v | email=%s", httpErr, user.Email)
		return "", NewHasuraError(
			"Failed to create user. Database connection error.",
			"Please try again later.",
		)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Hasura InsertUser returned %d — email=%s body=%s", resp.StatusCode, user.Email, string(b))
		return "", NewHasuraError(
			"Failed to create user. Database request failed.",
			fmt.Sprintf("Unexpected status %d.", resp.StatusCode),
		)
	}

	var respData struct {
		Data struct {
			InsertAuthServiceUsersOne struct {
				ID string `json:"id"`
			} `json:"insert_auth_service_users_one"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.Unmarshal(b, &respData); err != nil {
		log.Printf("InsertUser decode error: %v | body=%s", err, string(b))
		return "", NewServerError("Failed to process server response.")
	}

	if len(respData.Errors) > 0 {
		log.Printf("InsertUser Hasura errors: %v", respData.Errors)
		return "", NewHasuraError("Could not create user.", fmt.Sprintf("%v", respData.Errors))
	}

	if respData.Data.InsertAuthServiceUsersOne.ID == "" {
		return "", NewServerError("No user ID returned from database.")
	}

	log.Printf("User created: %s (%s)", user.Email, respData.Data.InsertAuthServiceUsersOne.ID)
	return respData.Data.InsertAuthServiceUsersOne.ID, nil
}

// UpdateUserProvider links a new OAuth provider to an existing user
func UpdateUserProvider(cfg Config, userID, provider, providerID string) (string, *ServiceError) {
	return execHasuraUpdate(cfg, `
		mutation UpdateProvider($id: uuid!, $provider: String!, $provider_id: String) {
		  update_auth_service_users_by_pk(
		    pk_columns: {id: $id},
		    _set: {provider: $provider, provider_id: $provider_id}
		  ) { id }
		}`,
		map[string]interface{}{
			"id":          userID,
			"provider":    provider,
			"provider_id": providerID,
		},
		"UpdateProvider",
	)
}

// UpdateUserPassword sets a new password for an existing user
func UpdateUserPassword(cfg Config, userID, password string) (string, *ServiceError) {
	return execHasuraUpdate(cfg, `
		mutation UpdatePassword($id: uuid!, $password: String!) {
		  update_auth_service_users_by_pk(
		    pk_columns: {id: $id},
		    _set: {password: $password}
		  ) { id }
		}`,
		map[string]interface{}{
			"id":       userID,
			"password": password,
		},
		"UpdatePassword",
	)
}

// UpdateUserPasswordAndProvider sets both password and provider for an existing user
func UpdateUserPasswordAndProvider(cfg Config, userID, password, provider string) (string, *ServiceError) {
	return execHasuraUpdate(cfg, `
		mutation UpdatePasswordAndProvider($id: uuid!, $password: String, $provider: String) {
		  update_auth_service_users_by_pk(
		    pk_columns: {id: $id},
		    _set: {password: $password, provider: $provider}
		  ) { id }
		}`,
		map[string]interface{}{
			"id":       userID,
			"password": password,
			"provider": provider,
		},
		"UpdatePasswordAndProvider",
	)
}

// execHasuraUpdate is a shared helper for simple by-pk update mutations
func execHasuraUpdate(cfg Config, query string, variables map[string]interface{}, opName string) (string, *ServiceError) {
	payload := map[string]interface{}{"query": query, "variables": variables}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, httpErr := http.DefaultClient.Do(req)
	if httpErr != nil {
		log.Printf("Hasura %s request error: %v", opName, httpErr)
		return "", NewHasuraError("Database connection error.", "Please try again later.")
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("Hasura %s returned %d — body=%s", opName, resp.StatusCode, string(b))
		return "", NewHasuraError("Database request failed.", fmt.Sprintf("Status %d.", resp.StatusCode))
	}

	var respData struct {
		Data struct {
			UpdateAuthServiceUsersByPk struct {
				ID string `json:"id"`
			} `json:"update_auth_service_users_by_pk"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.Unmarshal(b, &respData); err != nil {
		log.Printf("%s decode error: %v | body=%s", opName, err, string(b))
		return "", NewServerError("Failed to process server response.")
	}

	if len(respData.Errors) > 0 {
		log.Printf("%s Hasura errors: %v", opName, respData.Errors)
		return "", NewHasuraError("Database operation failed.", fmt.Sprintf("%v", respData.Errors))
	}

	return respData.Data.UpdateAuthServiceUsersByPk.ID, nil
}

// GetUserByEmail fetches a user from Hasura by email
func GetUserByEmail(cfg Config, email string) (models.User, *ServiceError) {
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

	payload := map[string]interface{}{"query": query, "variables": map[string]interface{}{"email": email}}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-hasura-admin-secret", cfg.HasuraAdminSecret)

	resp, httpErr := http.DefaultClient.Do(req)
	if httpErr != nil {
		log.Printf("GetUserByEmail request error: %v | email=%s", httpErr, email)
		return models.User{}, NewHasuraError("Database connection error.", "Please try again later.")
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		log.Printf("GetUserByEmail returned %d — email=%s body=%s", resp.StatusCode, email, string(b))
		return models.User{}, NewHasuraError("Database request failed.", fmt.Sprintf("Status %d.", resp.StatusCode))
	}

	var respData struct {
		Data struct {
			AuthServiceUsers []models.User `json:"auth_service_users"`
		} `json:"data"`
		Errors []interface{} `json:"errors"`
	}

	if err := json.Unmarshal(b, &respData); err != nil {
		log.Printf("GetUserByEmail decode error: %v | body=%s", err, string(b))
		return models.User{}, NewServerError("Failed to process server response.")
	}

	if len(respData.Errors) > 0 {
		log.Printf("GetUserByEmail Hasura errors: %v", respData.Errors)
		return models.User{}, NewHasuraError("Database query failed.", fmt.Sprintf("%v", respData.Errors))
	}

	if len(respData.Data.AuthServiceUsers) == 0 {
		return models.User{}, NewNotFoundError("User not found.")
	}

	return respData.Data.AuthServiceUsers[0], nil
}

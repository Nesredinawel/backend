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

// UpsertUserInHasura inserts or updates a user record in Hasura
func UpsertUserInHasura(cfg Config, user models.User) (string, error) {
	query := `
	mutation UpsertUser($email: String!, $name: String, $avatar_url: String) {
	  insert_users_one(
	    object: {email: $email, name: $name, avatar_url: $avatar_url},
	    on_conflict: {constraint: users_email_key, update_columns: [name, avatar_url]}
	  ) {
	    id
	  }
	}`

	payload := map[string]interface{}{
		"query": query,
		"variables": map[string]interface{}{
			"email":      user.Email,
			"name":       user.Name,
			"avatar_url": user.AvatarURL,
		},
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", cfg.HasuraURL, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
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
		Errors []map[string]interface{} `json:"errors"`
		Data   struct {
			InsertUsersOne struct {
				ID string `json:"id"`
			} `json:"insert_users_one"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", err
	}

	if len(respData.Errors) > 0 {
		errBytes, _ := json.MarshalIndent(respData.Errors, "", "  ")
		return "", fmt.Errorf("hasura returned errors: %s", string(errBytes))
	}

	if respData.Data.InsertUsersOne.ID == "" {
		return "", errors.New("no user id returned from hasura")
	}

	return respData.Data.InsertUsersOne.ID, nil
}

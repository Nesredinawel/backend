package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"

	"auth-service/models"
)

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

	req, err := http.NewRequest("POST", cfg.HasuraEndpoint, bytes.NewBuffer(body))
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

	var respData struct {
		Errors []interface{} `json:"errors"`
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
		return "", errors.New("hasura returned errors on upsert")
	}

	if respData.Data.InsertUsersOne.ID == "" {
		return "", errors.New("no user id returned from hasura")
	}

	return respData.Data.InsertUsersOne.ID, nil
}

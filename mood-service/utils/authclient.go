package utils

import (
	"fmt"
	"net/http"
)

func VerifyUserWithAuthService(cfg Config, token string) error {
	req, _ := http.NewRequest("GET", cfg.AuthServiceURL+"/user/profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("auth-service request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth-service returned status %d", resp.StatusCode)
	}
	return nil
}

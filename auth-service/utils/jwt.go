package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateJWT creates a Hasura-compatible JWT for a given user ID
func GenerateJWT(cfg Config, userID string) (string, error) {
	now := time.Now().UTC()

	claims := jwt.MapClaims{
		"sub": userID,
		"iss": "auth-service",
		"iat": now.Unix(),
		"exp": now.Add(24 * time.Hour).Unix(),
		"https://hasura.io/jwt/claims": map[string]interface{}{
			"x-hasura-default-role":  "user",
			"x-hasura-allowed-roles": []string{"user", "admin"},
			"x-hasura-user-id":       userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// JWTSession represents a complete token session (access + refresh)
type JWTSession struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// GenerateJWT creates Hasura-compatible access & refresh tokens
func GenerateJWT(cfg Config, userID, role string) (*JWTSession, error) {
	if role == "" {
		role = "user" // default fallback
	}

	now := time.Now().UTC()
	accessExp := now.Add(15 * time.Minute)   // short-lived access token
	refreshExp := now.Add(7 * 24 * time.Hour) // 7 days

	jti := uuid.New().String() // unique token ID for session traceability

	// --- Access Token (Hasura-compatible)
	accessClaims := jwt.MapClaims{
		"sub": userID,
		"iss": "auth-service",
		"aud": "hasura-backend",
		"iat": now.Unix(),
		"exp": accessExp.Unix(),
		"jti": jti,
		"https://hasura.io/jwt/claims": map[string]interface{}{
			"x-hasura-default-role":  role,
			"x-hasura-allowed-roles": []string{"user", "admin"},
			"x-hasura-user-id":       userID,
		},
	}

	// --- Refresh Token (used for new access tokens)
	refreshClaims := jwt.MapClaims{
		"sub": userID,
		"iss": "auth-service",
		"aud": "hasura-backend",
		"iat": now.Unix(),
		"exp": refreshExp.Unix(),
		"jti": jti,
		"type": "refresh", // helps distinguish refresh tokens
	}

	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return nil, err
	}

	return &JWTSession{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessExp.Sub(now).Seconds()),
	}, nil
}

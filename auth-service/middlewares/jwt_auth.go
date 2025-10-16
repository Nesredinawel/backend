package middlewares

import (
	"context"
	"net/http"
	"strings"

	"auth-service/utils"

	"github.com/golang-jwt/jwt/v4"
)

// JWTAuth middleware verifies the user's JWT token before allowing access
func JWTAuth(cfg utils.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "missing authorization header", http.StatusUnauthorized)
				return
			}

			// Expect "Bearer <token>"
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "invalid authorization format", http.StatusUnauthorized)
				return
			}

			// Parse and validate JWT
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Verify token signing method
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil {
				http.Error(w, "invalid or expired token", http.StatusUnauthorized)
				return
			}

			// Extract user_id from Hasura JWT claims
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
				claimsMap, ok := claims["https://hasura.io/jwt/claims"].(map[string]interface{})
				if !ok {
					http.Error(w, "invalid token payload", http.StatusUnauthorized)
					return
				}

				userID, ok := claimsMap["x-hasura-user-id"].(string)
				if !ok || userID == "" {
					http.Error(w, "invalid token payload", http.StatusUnauthorized)
					return
				}

				// Add user_id to request context
				ctx := context.WithValue(r.Context(), "user_id", userID)
				next.ServeHTTP(w, r.WithContext(ctx))
			} else {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		})
	}
}

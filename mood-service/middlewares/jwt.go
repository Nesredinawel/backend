package middlewares

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"mood-service/utils"

	"github.com/golang-jwt/jwt/v4"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

func JWTAuth(cfg utils.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Missing authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				http.Error(w, "Invalid token format", http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, "Invalid token claims", http.StatusUnauthorized)
				return
			}

			hasuraClaims, ok := claims["https://hasura.io/jwt/claims"].(map[string]interface{})
			if !ok {
				http.Error(w, "Invalid token payload", http.StatusUnauthorized)
				return
			}

			userID, ok := hasuraClaims["x-hasura-user-id"].(string)
			if !ok || userID == "" {
				http.Error(w, "Invalid token: missing user id", http.StatusUnauthorized)
				return
			}

			role, _ := hasuraClaims["x-hasura-default-role"].(string)

			ctx := context.WithValue(r.Context(), CtxUserID, userID)
			ctx = context.WithValue(ctx, CtxRole, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

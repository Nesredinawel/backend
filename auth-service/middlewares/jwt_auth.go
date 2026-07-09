package middlewares

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"auth-service/utils"

	"github.com/golang-jwt/jwt/v5"
)

func JWTAuth(cfg utils.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				utils.WriteJSONError(w, utils.NewAuthError("Missing authorization header"), http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid authorization format. Expected 'Bearer <token>'"), http.StatusUnauthorized)
				return
			}

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil {
				log.Printf("[JWT] Token parse error: %v\n", err)
				utils.WriteJSONError(w, utils.NewAuthError("Invalid or expired token. Please log in again."), http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				log.Println("[JWT] Token invalid or expired")
				utils.WriteJSONError(w, utils.NewAuthError("Invalid or expired token. Please log in again."), http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token claims"), http.StatusUnauthorized)
				return
			}

			hasuraClaimsRaw, ok := claims["https://hasura.io/jwt/claims"]
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token payload"), http.StatusUnauthorized)
				return
			}

			hasuraClaims, ok := hasuraClaimsRaw.(map[string]interface{})
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token payload"), http.StatusUnauthorized)
				return
			}

			userID, _ := hasuraClaims["x-hasura-user-id"].(string)
			if userID == "" {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token payload"), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), "user_id", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

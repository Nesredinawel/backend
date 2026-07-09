package middlewares

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"blog-service/utils"

	"github.com/golang-jwt/jwt/v5"
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
				utils.WriteJSONError(w, utils.NewAuthError("Missing authorization header"), http.StatusUnauthorized)
				log.Println("[JWT] Missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid authorization format. Expected 'Bearer <token>'"), http.StatusUnauthorized)
				log.Printf("[JWT] Invalid header format: %s\n", authHeader)
				return
			}
			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil {
				utils.WriteJSONError(w, utils.NewAuthError(fmt.Sprintf("Token parse error: %v", err)), http.StatusUnauthorized)
				log.Printf("[JWT] Token parse error: %v\n", err)
				return
			}

			if !token.Valid {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid or expired token. Please log in again."), http.StatusUnauthorized)
				log.Println("[JWT] Token invalid")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token claims"), http.StatusUnauthorized)
				log.Println("[JWT] Token claims not MapClaims")
				return
			}

			if exp, ok := claims["exp"].(float64); ok {
				if time.Now().UTC().After(time.Unix(int64(exp), 0)) {
					utils.WriteJSONError(w, utils.NewAuthError("Token expired. Please log in again."), http.StatusUnauthorized)
					log.Println("[JWT] Token expired")
					return
				}
			}

			hasuraClaimsRaw, ok := claims["https://hasura.io/jwt/claims"]
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Missing Hasura claims in token"), http.StatusUnauthorized)
				log.Println("[JWT] Missing Hasura claims")
				return
			}

			hasuraClaims, ok := hasuraClaimsRaw.(map[string]interface{})
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid Hasura claims format in token"), http.StatusUnauthorized)
				log.Println("[JWT] Hasura claims not map[string]interface{}")
				return
			}

			userID, _ := hasuraClaims["x-hasura-user-id"].(string)
			role, _ := hasuraClaims["x-hasura-default-role"].(string)

			if userID == "" {
				utils.WriteJSONError(w, utils.NewAuthError("Missing user ID in token"), http.StatusUnauthorized)
				log.Println("[JWT] x-hasura-user-id missing")
				return
			}

			ctx := context.WithValue(r.Context(), CtxUserID, userID)
			ctx = context.WithValue(ctx, CtxRole, role)

			log.Printf("[JWT] valid token for user %s role %s\n", userID, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		roleVal := r.Context().Value(CtxRole)
		role := ""
		if roleVal != nil {
			role = roleVal.(string)
		}
		if role != "admin" {
			utils.WriteJSONError(w, utils.NewAuthError("Admin role required to perform this action"), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

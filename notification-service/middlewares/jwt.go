package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"notification-service/utils"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

func JWTAuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
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
			tokenStr := parts[1]

			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(jwtSecret), nil
			})
			if err != nil {
				utils.WriteJSONError(w, utils.NewAuthError(fmt.Sprintf("Token parse error: %v", err)), http.StatusUnauthorized)
				log.Printf("[JWT] Token parse error: %v\n", err)
				return
			}

			if !token.Valid {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid or expired token. Please log in again."), http.StatusUnauthorized)
				log.Println("[JWT] Token invalid or expired")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token claims"), http.StatusUnauthorized)
				log.Println("[JWT] Token claims not MapClaims")
				return
			}

			exp, ok := claims["exp"].(float64)
			if !ok {
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token: missing expiration"), http.StatusUnauthorized)
				log.Println("[JWT] 'exp' claim missing or invalid")
				return
			}
			if time.Unix(int64(exp), 0).Before(time.Now()) {
				utils.WriteJSONError(w, utils.NewAuthError("Token expired. Please log in again."), http.StatusUnauthorized)
				log.Printf("[JWT] Token expired at %v", time.Unix(int64(exp), 0))
				return
			}

			var userID, role string

			if uid, ok := claims["user_id"].(string); ok && uid != "" {
				userID = uid
			}

			if userID == "" {
				rawClaims, ok := claims["https://hasura.io/jwt/claims"]
				if ok {
					if hmap, ok := rawClaims.(map[string]interface{}); ok {
						if uid, ok := hmap["x-hasura-user-id"].(string); ok && uid != "" {
							userID = uid
						}
						if r, ok := hmap["x-hasura-default-role"].(string); ok {
							role = r
						}
					}
				}
			}

			if userID == "" {
				log.Printf("[JWT] Could not extract user_id from token claims")
				utils.WriteJSONError(w, utils.NewAuthError("Invalid token payload: missing user ID"), http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), CtxUserID, userID)
			ctx = context.WithValue(ctx, CtxRole, role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserIDFromContext(r *http.Request) string {
	id, _ := r.Context().Value(CtxUserID).(string)
	return id
}

func GetRoleFromContext(r *http.Request) string {
	role, _ := r.Context().Value(CtxRole).(string)
	return role
}

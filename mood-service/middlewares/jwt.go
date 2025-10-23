package middlewares

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"mood-service/utils"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

type JSONError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// JWTAuth validates JWT tokens and injects user info into request context
func JWTAuth(cfg utils.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeJSONError(w, "Missing authorization header", http.StatusUnauthorized)
				log.Println("[JWT] Missing Authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeJSONError(w, "Invalid authorization header format", http.StatusUnauthorized)
				log.Printf("[JWT] Invalid header format: %s\n", authHeader)
				return
			}
			tokenString := parts[1]

			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(cfg.JWTSecret), nil
			})
			if err != nil {
				writeJSONError(w, fmt.Sprintf("Token parse error: %v", err), http.StatusUnauthorized)
				log.Printf("[JWT] Token parse error: %v\n", err)
				return
			}

			if !token.Valid {
				writeJSONError(w, "Invalid or expired token", http.StatusUnauthorized)
				log.Println("[JWT] Token invalid or expired")
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				writeJSONError(w, "Invalid token claims", http.StatusUnauthorized)
				log.Println("[JWT] Token claims not MapClaims")
				return
			}

			hasuraClaimsRaw, ok := claims["https://hasura.io/jwt/claims"]
			if !ok {
				writeJSONError(w, "Missing Hasura claims", http.StatusUnauthorized)
				log.Println("[JWT] Hasura claims missing")
				return
			}

			hasuraClaims, ok := hasuraClaimsRaw.(map[string]interface{})
			if !ok {
				writeJSONError(w, "Invalid Hasura claim format", http.StatusUnauthorized)
				log.Println("[JWT] Hasura claims format invalid")
				return
			}

			userID, _ := hasuraClaims["x-hasura-user-id"].(string)
			role, _ := hasuraClaims["x-hasura-default-role"].(string)

			if userID == "" {
				writeJSONError(w, "Missing user ID in token", http.StatusUnauthorized)
				log.Println("[JWT] x-hasura-user-id missing")
				return
			}

			// Inject into context
			ctx := context.WithValue(r.Context(), CtxUserID, userID)
			ctx = context.WithValue(ctx, CtxRole, role)

			log.Printf("[JWT] Valid token for user: %s, role: %s\n", userID, role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(JSONError{
		Success: false,
		Error:   message,
	})
}

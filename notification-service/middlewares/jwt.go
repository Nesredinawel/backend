package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"notification-service/config"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const (
	CtxUserID ctxKey = "user_id"
	CtxRole   ctxKey = "role"
)

// JSONError is the structure for error responses
type JSONError struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// JWTAuthMiddleware validates JWT and injects user info into context
func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeJSONError(w, "Missing Authorization header", http.StatusUnauthorized)
			log.Println("[JWT] Missing Authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeJSONError(w, "Invalid Authorization header format", http.StatusUnauthorized)
			log.Printf("[JWT] Invalid header format: %s\n", authHeader)
			return
		}
		tokenStr := parts[1]

		cfg := config.LoadConfig()
		token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
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

		// Validate expiration
		exp, ok := claims["exp"].(float64)
		if !ok {
			writeJSONError(w, "'exp' claim missing or invalid", http.StatusUnauthorized)
			log.Println("[JWT] 'exp' claim missing or invalid")
			return
		}
		if time.Unix(int64(exp), 0).Before(time.Now()) {
			writeJSONError(w, "Token expired", http.StatusUnauthorized)
			log.Printf("[JWT] Token expired at %v", time.Unix(int64(exp), 0))
			return
		}

		// Extract user ID and role
		var userID, role string

		if uid, ok := claims["user_id"].(string); ok && uid != "" {
			userID = uid
			log.Printf("[JWT] user_id found in claims: %s", userID)
		}

		if userID == "" {
			rawClaims, ok := claims["https://hasura.io/jwt/claims"]
			if ok {
				log.Printf("[JWT] Found Hasura claims, type=%T", rawClaims)
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
			log.Printf("[JWT] Could not extract user_id from token claims. Full claims: %+v", claims)
			writeJSONError(w, "Invalid token payload", http.StatusUnauthorized)
			return
		}

		// Inject into context
		ctx := context.WithValue(r.Context(), CtxUserID, userID)
		ctx = context.WithValue(ctx, CtxRole, role)
		log.Printf("[JWT] Valid token for user: %s, role: %s", userID, role)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Helper to extract user ID from context
func GetUserIDFromContext(r *http.Request) string {
	id, _ := r.Context().Value(CtxUserID).(string)
	return id
}

// Helper to extract role from context
func GetRoleFromContext(r *http.Request) string {
	role, _ := r.Context().Value(CtxRole).(string)
	return role
}

func writeJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(JSONError{
		Success: false,
		Error:   message,
	})
}

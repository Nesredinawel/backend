package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"auth-service/utils"

	"github.com/golang-jwt/jwt/v5"
)

func tokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func storeRefreshToken(userID, refreshToken string) {
	if utils.Rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "refresh:" + tokenHash(refreshToken)
	utils.Rdb.Set(ctx, key, userID, 7*24*time.Hour)
}

func isTokenBlacklisted(jti string) bool {
	if utils.Rdb == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	val, err := utils.Rdb.Get(ctx, "blacklist:"+jti).Result()
	return err == nil && val == "1"
}

func blacklistToken(jti string, exp int64) {
	if utils.Rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ttl := time.Until(time.Unix(exp, 0))
	if ttl > 0 {
		utils.Rdb.Set(ctx, "blacklist:"+jti, "1", ttl)
	}
}

func RefreshToken(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			RefreshToken string `json:"refresh_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeBadRequest(w, "Invalid request body.")
			return
		}
		if req.RefreshToken == "" {
			writeBadRequest(w, "refresh_token is required.")
			return
		}

		hash := tokenHash(req.RefreshToken)
		if utils.Rdb != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			userID, err := utils.Rdb.Get(ctx, "refresh:"+hash).Result()
			cancel()
			if err != nil {
				writeAuthError(w, "Invalid or expired refresh token. Please log in again.")
				return
			}
			if userID == "" {
				writeAuthError(w, "Invalid or expired refresh token. Please log in again.")
				return
			}
		}

		token, err := jwt.Parse(req.RefreshToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			writeAuthError(w, "Invalid or expired refresh token. Please log in again.")
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			writeAuthError(w, "Invalid token payload.")
			return
		}

		userID, _ := claims["sub"].(string)
		if userID == "" {
			writeAuthError(w, "Invalid token payload.")
			return
		}
		role, _ := claims["https://hasura.io/jwt/claims"].(map[string]interface{})
		roleStr := "user"
		if role != nil {
			if r, ok := role["x-hasura-default-role"].(string); ok {
				roleStr = r
			}
		}

		newSession, jwtErr := utils.GenerateJWT(cfg, userID, roleStr)
		if jwtErr != nil {
			writeServerError(w, "Failed to generate new tokens.")
			return
		}

		storeRefreshToken(userID, newSession.RefreshToken)

		writeSuccess(w, map[string]interface{}{
			"access_token":  newSession.AccessToken,
			"refresh_token": newSession.RefreshToken,
			"expires_in":    newSession.ExpiresIn,
		})
	}
}

func Logout(cfg utils.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			writeSuccess(w, map[string]interface{}{"message": "Logged out successfully."})
			return
		}

		tokenString := ""
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		}

		if tokenString != "" {
			token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
			if err == nil {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if jti, ok := claims["jti"].(string); ok {
						exp := int64(0)
						if expF, ok := claims["exp"].(float64); ok {
							exp = int64(expF)
						}
						blacklistToken(jti, exp)
					}
					if sub, ok := claims["sub"].(string); ok {
						deleteRefreshTokens(sub)
					}
				}
			}
		}

		writeSuccess(w, map[string]interface{}{"message": "Logged out successfully."})
	}
}

func deleteRefreshTokens(userID string) {
	if utils.Rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	iter := utils.Rdb.Scan(ctx, 0, "refresh:*", 100).Iterator()
	for iter.Next(ctx) {
		val, err := utils.Rdb.Get(ctx, iter.Val()).Result()
		if err == nil && val == userID {
			utils.Rdb.Del(ctx, iter.Val())
		}
	}
}

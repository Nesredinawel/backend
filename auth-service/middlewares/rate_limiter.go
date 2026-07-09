package middlewares

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"auth-service/utils"

	"github.com/redis/go-redis/v9"
)

type RateLimitConfig struct {
	Window time.Duration
	Limit  int
}

var defaultRateLimits = map[string]RateLimitConfig{
	"POST /auth/email/signup": {Window: 1 * time.Minute, Limit: 3},
	"POST /auth/email/login":  {Window: 1 * time.Minute, Limit: 10},
	"GET /auth/email/verify":  {Window: 1 * time.Minute, Limit: 10},
	"GET /auth/google/login":  {Window: 1 * time.Minute, Limit: 10},
}

func RateLimiter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if utils.Rdb == nil {
			next.ServeHTTP(w, r)
			return
		}

		routeKey := r.Method + " " + r.URL.Path
		cfg, ok := defaultRateLimits[routeKey]
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ip := r.RemoteAddr
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.Split(fwd, ",")[0]
		}

		key := "ratelimit:" + routeKey + ":" + ip
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		count, err := utils.Rdb.Incr(ctx, key).Result()
		if err != nil {
			log.Printf("Rate limiter error: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		if count == 1 {
			utils.Rdb.Expire(ctx, key, cfg.Window)
		}

		if count > int64(cfg.Limit) {
			ttl, _ := utils.Rdb.TTL(ctx, key).Result()
			w.Header().Set("Retry-After", ttl.String())
			utils.WriteJSONError(w, utils.NewServerError("Too many requests. Please try again later."), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func CheckAccountLockout(email string) bool {
	if utils.Rdb == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	val, err := utils.Rdb.Get(ctx, "lockout:"+email).Result()
	if err == redis.Nil {
		return false
	}
	if err != nil {
		log.Printf("Lockout check error: %v", err)
		return false
	}
	return val == "1"
}

func RecordFailedLogin(email string) {
	if utils.Rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	key := "login_attempts:" + email
	count, err := utils.Rdb.Incr(ctx, key).Result()
	if err != nil {
		log.Printf("Failed login tracking error: %v", err)
		return
	}
	if count == 1 {
		utils.Rdb.Expire(ctx, key, 15*time.Minute)
	}

	if count >= 10 {
		utils.Rdb.Set(ctx, "lockout:"+email, "1", 15*time.Minute)
		utils.Rdb.Del(ctx, key)
		log.Printf("Account locked due to failed attempts: %s", email)
	}
}

func ResetFailedLogins(email string) {
	if utils.Rdb == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	utils.Rdb.Del(ctx, "login_attempts:"+email)
	utils.Rdb.Del(ctx, "lockout:"+email)
}

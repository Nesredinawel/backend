package middlewares

import (
	"log"
	"net/http"
	"time"
)

// RequestLogger is a custom middleware for detailed logging
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[DEBUG] ➜ %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("[DEBUG] ⇦ %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
	})
}

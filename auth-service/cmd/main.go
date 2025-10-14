package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"auth-service/handlers"
	"auth-service/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// requestLogger is a simple custom middleware for more verbose route debugging
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[DEBUG] ➜ %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("[DEBUG] ⇦ %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	// Load configuration
	cfg, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Initialize Google OAuth once at startup
	handlers.InitGoogleOAuth(cfg)
	log.Println("✅ Google OAuth configuration initialized")

	// Setup router
	r := chi.NewRouter()

	// Built-in middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json", "text/plain"))
	r.Use(requestLogger) // Custom detailed logger

	// Health check route
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Google OAuth routes
	r.Get("/auth/google/login", handlers.GoogleLogin())
	r.Get("/auth/google/callback", handlers.GoogleCallback(cfg))

	// Email-based authentication routes
	r.Post("/auth/email/signup", handlers.EmailSignup(cfg))
	r.Post("/auth/email/login", handlers.EmailLogin(cfg))

	// 404 Debug Helper
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("❌ Route not found: %s %s", r.Method, r.URL.Path)
		log.Println(msg)
		http.Error(w, msg, http.StatusNotFound)
	})

	// Server startup
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	serverAddr := fmt.Sprintf("http://localhost:%s", port)
	log.Println("========================================")
	log.Printf("AUTH SERVICE RUNNING ON %s", serverAddr)
	log.Println("📡 Available routes:")
	log.Println("  → GET  /healthz")
	log.Println("  → GET  /auth/google/login")
	log.Println("  → GET  /auth/google/callback")
	log.Println("  → POST /auth/email/signup")
	log.Println("  → POST /auth/email/login")
	log.Println("========================================")

	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("❌ Server failed to start: %v", err)
	}
}

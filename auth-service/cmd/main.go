package main

import (
	"log"
	"net/http"
	"os"

	"auth-service/handlers"
	"auth-service/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	// Load configuration
	cfg, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("❌ failed to load config: %v", err)
	}

	// Initialize Google OAuth once on startup
	handlers.InitGoogleOAuth(cfg)

	// Setup router
	r := chi.NewRouter()

	// Middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json", "text/plain"))

	// Health check route
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Auth routes
	r.Get("/auth/google/login", handlers.GoogleLogin())
	r.Get("/auth/google/callback", handlers.GoogleCallback(cfg))

	// Server port setup
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("🚀 auth-service running on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("❌ server failed: %v", err)
	}
}

package routes

import (
	"fmt"
	"net/http"
	"os"

	"auth-service/handlers"
	"auth-service/middlewares"
	"auth-service/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRoutes(cfg utils.Config) http.Handler {
	r := chi.NewRouter()

	// Built-in middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json", "text/plain"))
	r.Use(middlewares.RequestLogger) // custom logger

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Google OAuth
	r.Get("/auth/google/login", handlers.GoogleLogin())
	r.Get("/auth/google/callback", handlers.GoogleCallback(cfg))

	// Email auth
	r.Post("/auth/email/signup", handlers.EmailSignup(cfg))
	r.Post("/auth/email/login", handlers.EmailLogin(cfg))

	// 404 Debug
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("❌ Route not found: %s %s", r.Method, r.URL.Path)
		http.Error(w, msg, http.StatusNotFound)
	})

	return r
}

func PrintRoutes(cfg utils.Config) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	serverAddr := fmt.Sprintf("http://localhost:%s", port)
	fmt.Println("========================================")
	fmt.Printf("AUTH SERVICE RUNNING ON %s\n", serverAddr)
	fmt.Println("📡 Available routes:")
	fmt.Println("  → GET  /healthz")
	fmt.Println("  → GET  /auth/google/login")
	fmt.Println("  → GET  /auth/google/callback")
	fmt.Println("  → POST /auth/email/signup")
	fmt.Println("  → POST /auth/email/login")
	fmt.Println("========================================")
}

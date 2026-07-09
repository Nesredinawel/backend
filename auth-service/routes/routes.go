package routes

import (
	"fmt"
	"net/http"
	"os"

	"auth-service/handlers"
	"auth-service/middlewares"
	"auth-service/utils"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func SetupRoutes(cfg utils.Config) http.Handler {
	r := chi.NewRouter()

	// ================================
	// ⚙️ Global Middlewares
	// ================================
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(middlewares.RequestLogger)
	r.Use(middlewares.RateLimiter)

	// ================================
	// 🩺 Health Check
	// ================================
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// ================================
	// 🔑 Public Authentication Routes
	// ================================
	r.Route("/auth", func(r chi.Router) {
		// Google OAuth
		r.Get("/google/login", handlers.GoogleLogin())
		r.Get("/google/callback", handlers.GoogleCallback(cfg))

		// Email/Password Auth
		r.Post("/email/signup", handlers.EmailSignup(cfg))
		r.Post("/email/login", handlers.EmailLogin(cfg))
		r.Get("/email/verify", handlers.EmailVerify(cfg))

		// Token management
		r.Post("/refresh", handlers.RefreshToken(cfg))
		r.Post("/logout", handlers.Logout(cfg))
	})

	// ================================
	// 🔒 Protected User Routes
	// ================================
	r.Group(func(r chi.Router) {
		r.Use(middlewares.JWTAuth(cfg))

		r.Get("/user/profile", handlers.GetUserProfile(cfg))
		r.Put("/user/profile", handlers.UpdateUserProfile(cfg))
		r.Post("/user/password/change", handlers.ChangePassword(cfg))
	})

	// ================================
	// ❌ 404 Not Found Handler
	// ================================
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSONError(w, utils.NewNotFoundError(fmt.Sprintf("Route not found: %s %s", r.Method, r.URL.Path)), http.StatusNotFound)
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
	fmt.Println("Routes:")
	fmt.Println("  GET  /healthz")
	fmt.Println("  GET  /auth/google/login")
	fmt.Println("  GET  /auth/google/callback")
	fmt.Println("  POST /auth/email/signup")
	fmt.Println("  POST /auth/email/login")
	fmt.Println("  GET  /auth/email/verify")
	fmt.Println("  POST /auth/refresh")
	fmt.Println("  POST /auth/logout")
	fmt.Println("  GET  /user/profile              (JWT)")
	fmt.Println("  PUT  /user/profile              (JWT)")
	fmt.Println("  POST /user/password/change      (JWT)")
	fmt.Println("========================================")
}

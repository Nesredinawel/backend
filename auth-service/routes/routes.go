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

	// ================================
	// ⚙️ Global Middlewares
	// ================================
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json", "text/plain"))
	r.Use(middlewares.RequestLogger)

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
	})

	// ================================
	// 🔒 Protected User Profile Routes
	// ================================
	r.Group(func(r chi.Router) {
		r.Use(middlewares.JWTAuth(cfg)) // ✅ only authenticated users can access

		r.Get("/user/profile", handlers.GetUserProfile(cfg))
		r.Put("/user/profile", handlers.UpdateUserProfile(cfg))
	})

	// ================================
	// ❌ 404 Not Found Handler
	// ================================
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("❌ Route not found: %s %s", r.Method, r.URL.Path)
		http.Error(w, msg, http.StatusNotFound)
	})

	return r
}

// ===================================
// 🖨️ Helper Function: Print Routes
// ===================================
func PrintRoutes(cfg utils.Config) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	serverAddr := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println("========================================")
	fmt.Printf("🚀 AUTH SERVICE RUNNING ON %s\n", serverAddr)
	fmt.Println("📡 Available routes:")
	fmt.Println("  → GET  /healthz")
	fmt.Println("  → GET  /auth/google/login")
	fmt.Println("  → GET  /auth/google/callback")
	fmt.Println("  → POST /auth/email/signup")
	fmt.Println("  → POST /auth/email/login")
	fmt.Println("  → GET  /user/profile         (🔒 JWT required)")
	fmt.Println("  → PUT  /user/profile         (🔒 JWT required)")
	fmt.Println("========================================")
}

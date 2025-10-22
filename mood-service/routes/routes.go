package routes

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"

	"mood-service/handlers"
	"mood-service/middlewares"
	"mood-service/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupRoutes sets up all routes for mood-service
func SetupRoutes(cfg utils.Config, db *sql.DB) http.Handler {
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
	// 🔒 Protected Mood Routes
	// ================================
	r.Group(func(r chi.Router) {
		r.Use(middlewares.JWTAuth(cfg)) // only authenticated users

		r.Post("/moods", handlers.CreateMood(cfg, db))
		r.Get("/moods", handlers.ListMoods(cfg, db))
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
		port = "8082" // default for mood-service
	}
	serverAddr := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println("========================================")
	fmt.Printf("🚀 MOOD SERVICE RUNNING ON %s\n", serverAddr)
	fmt.Println("📡 Available routes:")
	fmt.Println("  → GET  /healthz")
	fmt.Println("  → POST /moods           (🔒 JWT required)")
	fmt.Println("  → GET  /moods           (🔒 JWT required)")
	fmt.Println("========================================")
}

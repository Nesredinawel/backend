package routes

import (
	"fmt"
	"net/http"

	"mood-service/handlers"
	"mood-service/middlewares"
	"mood-service/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRoutes(cfg utils.Config) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json", "text/plain"))

	r.Route("/api/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.JWTAuth(cfg))

			r.Post("/moods", handlers.CreateMood(cfg))
			r.Get("/moods", handlers.GetMoods(cfg))
			r.Put("/moods/{id}", handlers.UpdateMood(cfg))

			// analytics
			r.Get("/moods/analytics", handlers.MoodKPI(cfg))
		})
	})

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		msg := fmt.Sprintf("❌ Route not found: %s %s", r.Method, r.URL.Path)
		http.Error(w, msg, http.StatusNotFound)
	})

	return r
}

func PrintRoutes(cfg utils.Config) {
	port := cfg.Port
	if port == "" {
		port = "8082"
	}
	serverAddr := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println("========================================")
	fmt.Printf("🚀 MOOD SERVICE RUNNING ON %s\n", serverAddr)
	fmt.Println("📡 Available routes:")
	fmt.Println("  → GET    /healthz")
	fmt.Println("  → POST   /api/v1/moods (🔒 JWT required)")
	fmt.Println("  → PUT    /api/v1/moods/{id} (🔒 JWT required)")
	fmt.Println("  → GET    /api/v1/moods (🔒 JWT required)")
	fmt.Println("========================================")
}

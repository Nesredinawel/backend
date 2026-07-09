package routes

import (
	"fmt"
	"net/http"

	"blog-service/handlers"
	"blog-service/middlewares"
	"blog-service/utils"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func SetupRoutes(cfg utils.Config) http.Handler {
	r := chi.NewRouter()

	// Global middlewares
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))

	// Health
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	// API v1
	r.Route("/api/v1", func(r chi.Router) {

		// Public endpoints (no JWT required)
		r.Get("/posts", handlers.GetPosts(cfg))
		r.Get("/posts/external", handlers.GetExternalPosts(cfg))
		r.Get("/posts/{id}", handlers.GetPost(cfg))

		// Protected endpoints (JWT required)
		r.Group(func(r chi.Router) {
			r.Use(middlewares.JWTAuth(cfg))

			// Admin-only endpoints
			r.Group(func(r chi.Router) {
				r.Use(middlewares.AdminOnly)
				r.Post("/posts", handlers.CreatePost(cfg))
				r.Put("/posts/{id}", handlers.UpdatePost(cfg))
				r.Delete("/posts/{id}", handlers.DeletePost(cfg))
			})
		})
	})

	// Not found / method not allowed
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSONError(w, utils.NewNotFoundError(fmt.Sprintf("Route not found: %s %s", r.Method, r.URL.Path)), http.StatusNotFound)
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		utils.WriteJSONError(w, utils.NewBadRequestError(fmt.Sprintf("Method not allowed: %s %s", r.Method, r.URL.Path)), http.StatusMethodNotAllowed)
	})

	return r
}

func PrintRoutes(cfg utils.Config) {
	port := cfg.Port
	if port == "" {
		port = "8083"
	}
	serverAddr := fmt.Sprintf("http://localhost:%s", port)

	fmt.Println("========================================")
	fmt.Printf("BLOG SERVICE RUNNING ON %s\n", serverAddr)
	fmt.Println("Available routes:")
	fmt.Println("  GET    /healthz")
	fmt.Println("  GET    /api/v1/posts            (?category=&limit=&offset=)")
	fmt.Println("  GET    /api/v1/posts/external   (?tag=&per_page=)")
	fmt.Println("  GET    /api/v1/posts/{id}")
	fmt.Println("  POST   /api/v1/posts            (JWT + admin only)")
	fmt.Println("  PUT    /api/v1/posts/{id}       (JWT + admin only)")
	fmt.Println("  DELETE /api/v1/posts/{id}       (JWT + admin only)")
	fmt.Println("========================================")
}

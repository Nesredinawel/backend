package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

// helper to create reverse proxy
func reverseProxy(target string) http.Handler {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)

	// preserve original headers (JWT!)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = url.Host
	}

	return proxy
}

func main() {
	r := chi.NewRouter()

	// ===============================
	// Global middlewares
	// ===============================
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// ===============================
	// Health check
	// ===============================
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("api-gateway ok"))
	})

	// ===============================
	// AUTH SERVICE (8081)
	// ===============================
	r.Route("/auth", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://auth-service:8081"))
	})

	r.Route("/user", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://auth-service:8081"))
	})

	// ===============================
	// MOOD SERVICE (8082)
	// ===============================
	r.Route("/api/v1/moods", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://mood-service:8082"))
	})

	r.Route("/api/v1/moods/analytics", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://mood-service:8082"))
	})

	// ===============================
	// BLOG SERVICE (8083)
	// ===============================
	r.Route("/api/v1/posts", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://blog-service:8083"))
	})

	// ===============================
	// NOTIFICATION SERVICE (8084)
	// ===============================
	r.Route("/notifications", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://notification-service:8084"))
	})

	r.Route("/ws", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://notification-service:8084"))
	})

	// ===============================
	// API DOCS SERVICE (8085)
	// ===============================
	r.Route("/docs", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://api-docs-service:8085"))
	})

	r.Route("/swagger", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://api-docs-service:8085"))
	})

	r.Route("/redoc", func(r chi.Router) {
		r.Mount("/", reverseProxy("http://api-docs-service:8085"))
	})

	// ===============================
	// CORS
	// ===============================
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // tighten in prod
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	log.Println("🚪 API Gateway running on :8081")
	log.Fatal(http.ListenAndServe(":8081", c.Handler(r)))
}

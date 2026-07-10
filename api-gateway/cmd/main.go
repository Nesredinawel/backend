package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
)

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// helper to create reverse proxy
func reverseProxy(target string) http.Handler {
	url, _ := url.Parse(target)
	proxy := httputil.NewSingleHostReverseProxy(url)

	// preserve original headers (JWT!) and forward the real host/proto
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		if req.Header.Get("X-Forwarded-Host") == "" {
			req.Header.Set("X-Forwarded-Host", req.Host)
		}
		if req.Header.Get("X-Forwarded-Proto") == "" {
			req.Header.Set("X-Forwarded-Proto", "https")
		}
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
	authTarget := getEnv("AUTH_SERVICE_URL", "http://auth-service:8081")
	r.HandleFunc("/auth/*", reverseProxy(authTarget))
	r.HandleFunc("/auth", reverseProxy(authTarget))

	r.HandleFunc("/user/*", reverseProxy(authTarget))
	r.HandleFunc("/user", reverseProxy(authTarget))

	// ===============================
	// MOOD SERVICE (8082)
	// ===============================
	moodTarget := getEnv("MOOD_SERVICE_URL", "http://mood-service:8082")
	r.HandleFunc("/api/v1/moods/*", reverseProxy(moodTarget))
	r.HandleFunc("/api/v1/moods", reverseProxy(moodTarget))

	// ===============================
	// BLOG SERVICE (8083)
	// ===============================
	blogTarget := getEnv("BLOG_SERVICE_URL", "http://blog-service:8083")
	r.HandleFunc("/api/v1/posts/*", reverseProxy(blogTarget))
	r.HandleFunc("/api/v1/posts", reverseProxy(blogTarget))

	// ===============================
	// NOTIFICATION SERVICE (8084)
	// ===============================
	notifTarget := getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8084")
	r.HandleFunc("/notifications/*", reverseProxy(notifTarget))
	r.HandleFunc("/notifications", reverseProxy(notifTarget))

	r.HandleFunc("/ws/*", reverseProxy(notifTarget))
	r.HandleFunc("/ws", reverseProxy(notifTarget))

	// ===============================
	// API DOCS SERVICE (8085)
	// ===============================
	docsTarget := getEnv("API_DOCS_SERVICE_URL", "http://api-docs-service:8085")
	r.HandleFunc("/docs/*", reverseProxy(docsTarget))
	r.HandleFunc("/docs", reverseProxy(docsTarget))

	r.HandleFunc("/swagger/*", reverseProxy(docsTarget))
	r.HandleFunc("/swagger", reverseProxy(docsTarget))

	r.HandleFunc("/redoc/*", reverseProxy(docsTarget))
	r.HandleFunc("/redoc", reverseProxy(docsTarget))

	// ===============================
	// CORS
	// ===============================
	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: allowedOrigin != "*",
	})

	log.Println("🚪 API Gateway running on :8081")
	log.Fatal(http.ListenAndServe(":8081", c.Handler(r)))
}

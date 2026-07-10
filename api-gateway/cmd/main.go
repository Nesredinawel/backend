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
	// Health check & root
	// ===============================
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("api-gateway ok"))
	})

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	})

	// ===============================
	// AUTH SERVICE (8081)
	// ===============================
	authTarget := getEnv("AUTH_SERVICE_URL", "http://auth-service:8081")
	r.Handle("/auth/*", reverseProxy(authTarget))
	r.Handle("/auth", reverseProxy(authTarget))

	r.Handle("/user/*", reverseProxy(authTarget))
	r.Handle("/user", reverseProxy(authTarget))

	// ===============================
	// MOOD SERVICE (8082)
	// ===============================
	moodTarget := getEnv("MOOD_SERVICE_URL", "http://mood-service:8082")
	r.Handle("/api/v1/moods/*", reverseProxy(moodTarget))
	r.Handle("/api/v1/moods", reverseProxy(moodTarget))

	// ===============================
	// BLOG SERVICE (8083)
	// ===============================
	blogTarget := getEnv("BLOG_SERVICE_URL", "http://blog-service:8083")
	r.Handle("/api/v1/posts/*", reverseProxy(blogTarget))
	r.Handle("/api/v1/posts", reverseProxy(blogTarget))

	// ===============================
	// NOTIFICATION SERVICE (8084)
	// ===============================
	notifTarget := getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8084")
	r.Handle("/notifications/*", reverseProxy(notifTarget))
	r.Handle("/notifications", reverseProxy(notifTarget))

	r.Handle("/ws/*", reverseProxy(notifTarget))
	r.Handle("/ws", reverseProxy(notifTarget))

	// ===============================
	// API DOCS SERVICE (8085)
	// ===============================
	docsTarget := getEnv("API_DOCS_SERVICE_URL", "http://api-docs-service:8085")
	r.Handle("/docs/*", reverseProxy(docsTarget))
	r.Handle("/docs", reverseProxy(docsTarget))

	r.Handle("/swagger/*", reverseProxy(docsTarget))
	r.Handle("/swagger", reverseProxy(docsTarget))

	r.Handle("/redoc/*", reverseProxy(docsTarget))
	r.Handle("/redoc", reverseProxy(docsTarget))

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

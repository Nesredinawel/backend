package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"

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
	// Target URLs
	authTarget := getEnv("AUTH_SERVICE_URL", "http://auth-service:8081")
	moodTarget := getEnv("MOOD_SERVICE_URL", "http://mood-service:8082")
	blogTarget := getEnv("BLOG_SERVICE_URL", "http://blog-service:8083")
	notifTarget := getEnv("NOTIFICATION_SERVICE_URL", "http://notification-service:8084")
	docsTarget := getEnv("API_DOCS_SERVICE_URL", "http://api-docs-service:8085")

	// Build proxy handlers
	authProxy := reverseProxy(authTarget)
	moodProxy := reverseProxy(moodTarget)
	blogProxy := reverseProxy(blogTarget)
	notifProxy := reverseProxy(notifTarget)
	docsProxy := reverseProxy(docsTarget)

	// Chi router handles healthz, root, and unmatched routes
	chiRouter := chi.NewRouter()

	chiRouter.Use(middleware.RequestID)
	chiRouter.Use(middleware.RealIP)
	chiRouter.Use(middleware.Logger)
	chiRouter.Use(middleware.Recoverer)

	chiRouter.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("api-gateway ok"))
	})

	chiRouter.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusFound)
	})

	// ===============================
	// Main handler: direct prefix routing for all proxy paths
	// (bypasses chi's pattern matching to avoid path-stripping issues)
	// ===============================
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		switch {
		case strings.HasPrefix(path, "/auth") || strings.HasPrefix(path, "/user"):
			authProxy.ServeHTTP(w, r)
		case strings.HasPrefix(path, "/api/v1/moods"):
			moodProxy.ServeHTTP(w, r)
		case strings.HasPrefix(path, "/api/v1/posts"):
			blogProxy.ServeHTTP(w, r)
		case strings.HasPrefix(path, "/notifications") || strings.HasPrefix(path, "/ws"):
			notifProxy.ServeHTTP(w, r)
		case strings.HasPrefix(path, "/docs") || strings.HasPrefix(path, "/swagger") || strings.HasPrefix(path, "/redoc"):
			docsProxy.ServeHTTP(w, r)
		default:
			// Let chi handle /healthz, /, and anything else
			chiRouter.ServeHTTP(w, r)
		}
	})

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
	log.Fatal(http.ListenAndServe(":8081", c.Handler(mainHandler)))
}

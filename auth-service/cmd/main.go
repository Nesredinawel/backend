package main

import (
	"log"
	"net/http"
	"os"

	"auth-service/handlers"
	"auth-service/routes"
	"auth-service/utils"

	"github.com/rs/cors"
)

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		if r.Header.Get("X-Forwarded-Proto") == "http" {
			if r.Method == "GET" || r.Method == "HEAD" {
				http.Redirect(w, r, "https://"+r.Host+r.RequestURI, http.StatusMovedPermanently)
				return
			}
			writeHTTPSRequired(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeHTTPSRequired(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(`{"success":false,"error":"HTTPS required","code":"BAD_REQUEST"}`))
}

func main() {
	cfg, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	handlers.InitGoogleOAuth(cfg)
	log.Println("Google OAuth initialized")

	utils.InitHasura(cfg)
	log.Println("Hasura client initialized")

	utils.InitRedis(cfg)

	r := routes.SetupRoutes(cfg)

	routes.PrintRoutes(cfg)

	// Security headers middleware
	r = securityHeaders(r)

	allowedOrigin := os.Getenv("CORS_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "http://localhost:8081"
	}
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{allowedOrigin},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: allowedOrigin != "*",
	})
	handler := c.Handler(r)

	port := cfg.Port
	log.Printf("Auth service running on :%s", port)
	log.Fatal(http.ListenAndServe("127.0.0.1:"+port, handler))
}

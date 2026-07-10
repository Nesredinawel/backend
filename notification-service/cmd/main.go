package main

import (
	"log"
	"net/http"
	"os"

	"notification-service/config"
	"notification-service/handlers"
	"notification-service/routes"
	"notification-service/services"
	"notification-service/utils"

	"github.com/go-chi/chi/v5"
	"github.com/rs/cors"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize Redis connection
	utils.InitRedis(cfg.RedisAddr)

	// Initialize notification manager (stores last 100 per user)
	services.InitManager(100)

	// Define all channels that microservices will publish to
	channels := []string{
		"auth_events",          // events from auth-service (signup, login, etc.)
		"mood_events",          // events from mood-service (add/update mood)
		"blog_events",          // events from blog-service (new blog post, etc.)
		"global_notifications", // optional fallback for general system messages
	}

	// Start Redis listener for all incoming events concurrently
	go handlers.StartRedisListener(channels...)

	// Setup HTTP routes
	r := chi.NewRouter()
	routes.RegisterRoutes(r)

	// CORS
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

	// Start HTTP server
	addr := ":" + cfg.Port
	log.Printf("🚀 notification-service running on %s", addr)
	log.Printf("📡 Listening to Redis channels: %v", channels)

	log.Fatal(http.ListenAndServe(addr, c.Handler(r)))
}

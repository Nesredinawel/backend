package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"blog-service/routes"
	"blog-service/utils"

	"github.com/rs/cors"
)

func main() {
	// Load configuration (automatically initializes Redis)
	cfg := utils.LoadConfig()

	// Start background Dev.to article poller
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go utils.StartExternalFetcher(ctx)

	// Setup routes
	r := routes.SetupRoutes(cfg)

	// Print routes info
	routes.PrintRoutes(cfg)

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

	// Start server
	log.Printf("[blog-service] ✅ Starting server on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, c.Handler(r)))
}

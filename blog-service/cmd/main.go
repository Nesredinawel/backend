package main

import (
	"context"
	"log"
	"net/http"

	"blog-service/routes"
	"blog-service/utils"
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

	// Start server
	log.Printf("[blog-service] ✅ Starting server on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

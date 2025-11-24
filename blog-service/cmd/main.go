package main

import (
	"log"
	"net/http"

	"blog-service/routes"
	"blog-service/utils"
)

func main() {
	// Load configuration (automatically initializes Redis)
	cfg := utils.LoadConfig()

	// Setup routes
	r := routes.SetupRoutes(cfg)

	// Print routes info
	routes.PrintRoutes(cfg)

	// Start server
	log.Printf("[mood-service] ✅ Starting server on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

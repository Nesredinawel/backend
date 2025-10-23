package main

import (
	"log"
	"net/http"

	"mood-service/routes"
	"mood-service/utils"
)

func main() {
	// Load configuration
	cfg, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Setup routes
	r := routes.SetupRoutes(cfg)

	// Print routes info
	routes.PrintRoutes(cfg)

	// Start server
	port := cfg.Port
	log.Printf("[mood-service] ✅ Starting server on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

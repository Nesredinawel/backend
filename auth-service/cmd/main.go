package main

import (
	"log"
	"net/http"

	"auth-service/routes"
	"auth-service/utils"
	"auth-service/handlers"
)

func main() {
	// Load configuration
	cfg, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Initialize Google OAuth
	handlers.InitGoogleOAuth(cfg)
	log.Println("✅ Google OAuth configuration initialized")

	// Setup routes
	r := routes.SetupRoutes(cfg)

	// Print routes info
	routes.PrintRoutes(cfg)

	// Start server
	port := cfg.Port
	log.Fatal(http.ListenAndServe(":"+port, r))
}

package main

import (
	"log"
	"net/http"

	"auth-service/handlers"
	"auth-service/routes"
	"auth-service/utils"

	"github.com/rs/cors" // import CORS package
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

	// -------------------------------
	// Wrap router with CORS middleware
	// -------------------------------
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081"}, // allow API Docs UI
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(r)

	// Start server
	port := cfg.Port
	log.Printf("🚀 Auth service running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

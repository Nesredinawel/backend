package main

import (
	"log"
	"net/http"

	"mood-service/routes"
	"mood-service/utils"

	"github.com/rs/cors"
)

func main() {
	// Load configuration (automatically initializes Redis)
	cfg := utils.LoadConfig()

	// Setup routes
	r := routes.SetupRoutes(cfg)

	// Print routes info
	routes.PrintRoutes(cfg)

	// -------------------------------
	// Wrap router with CORS middleware
	// -------------------------------
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:8081"}, // allow Swagger UI or frontend
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})
	handler := c.Handler(r)

	// Start server
	log.Printf("[mood-service] ✅ Starting server on port %s\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, handler))
}

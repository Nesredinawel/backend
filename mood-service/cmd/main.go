package main

import (
	"log"
	"net/http"
	"os"

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

	// Start server
	addr := ":" + cfg.Port
	if bind := os.Getenv("BIND_ADDR"); bind != "" {
		addr = bind + ":" + cfg.Port
	}
	log.Printf("[mood-service] ✅ Starting server on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}

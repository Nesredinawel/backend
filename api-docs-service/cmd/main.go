package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/rs/cors"

	"api-docs-service/handlers"
)

func main() {
	r := mux.NewRouter()

	// -------------------------------
	// Determine file paths inside container
	// -------------------------------
	baseDocsPath := "/root/docs" // Update this if using a different volume mount

	masterYAML := baseDocsPath + "/master.yaml"
	authYAML := baseDocsPath + "/auth-service.yaml"
	moodYAML := baseDocsPath + "/mood-service.yaml"
	blogYAML := baseDocsPath + "/blog-service.yaml"
	notificationYAML := baseDocsPath + "/notification-service.yaml"

	// Ensure files exist at startup
	files := []string{masterYAML, authYAML, moodYAML, blogYAML, notificationYAML}
	for _, f := range files {
		if _, err := os.Stat(f); err != nil {
			log.Printf("⚠️ WARNING: File not found: %s", f)
		}
	}

	// -------------------------------
	// Serve OpenAPI YAML files
	// -------------------------------
	r.HandleFunc("/docs/master.yaml", handlers.OpenAPISpec(masterYAML)).Methods("GET", "OPTIONS")
	r.HandleFunc("/docs/auth.yaml", handlers.OpenAPISpec(authYAML)).Methods("GET", "OPTIONS")
	r.HandleFunc("/docs/mood.yaml", handlers.OpenAPISpec(moodYAML)).Methods("GET", "OPTIONS")
	r.HandleFunc("/docs/blog.yaml", handlers.OpenAPISpec(blogYAML)).Methods("GET", "OPTIONS")
	r.HandleFunc("/docs/notifications.yaml", handlers.OpenAPISpec(notificationYAML)).Methods("GET", "OPTIONS")

	// -------------------------------
	// Serve Swagger UI (and Redoc if needed)
	// -------------------------------
	r.PathPrefix("/swagger/").Handler(handlers.SwaggerUI("/docs/master.yaml"))
	r.PathPrefix("/redoc/").Handler(handlers.RedocUI("/docs/master.yaml"))

	// -------------------------------
	// CORS setup
	// -------------------------------
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

	handler := c.Handler(r)

	// -------------------------------
	// Start server
	// -------------------------------
	addr := ":8085"
	if bind := os.Getenv("BIND_ADDR"); bind != "" {
		addr = bind + ":8085"
	}
	log.Printf("API Docs running on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

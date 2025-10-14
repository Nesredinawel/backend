package main

import (
	"log"
	"net/http"
	"os"

	"auth-service/handlers"
	"auth-service/utils"

	"github.com/go-chi/chi/v5"
)

func main() {
	cfg := utils.LoadConfig()

	r := chi.NewRouter()

	// Health check
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Get("/auth/google/login", handlers.GoogleLogin(cfg))
	r.Get("/auth/google/callback", handlers.GoogleCallback(cfg))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Printf("🚀 auth-service running on :%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

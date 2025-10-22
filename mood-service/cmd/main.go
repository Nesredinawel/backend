package main

import (
	"database/sql"
	"log"
	"mood-service/routes"
	"mood-service/utils"
	"net/http"
	"time"
)

func main() {
	cfg, _ := utils.LoadConfig()

	var db *sql.DB
	var err error

	// Retry loop for DB connection
	for i := 1; i <= 10; i++ {
		db, err = utils.ConnectDB(cfg.PostgresDSN)
		if err == nil {
			log.Println("✅ Connected to PostgreSQL")
			break
		}
		log.Printf("⏳ Waiting for database... (attempt %d/10): %v", i, err)
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		log.Fatalf("❌ Could not connect to PostgreSQL after 10 attempts: %v", err)
	}
	defer db.Close()

	r := routes.SetupRoutes(cfg, db)

	addr := ":" + cfg.Port
	log.Printf("🚀 mood-service running on %s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("❌ server failed: %v", err)
	}
}

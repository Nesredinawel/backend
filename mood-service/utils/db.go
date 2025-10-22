package utils

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

func ConnectDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	// optional ping
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	return db, nil
}

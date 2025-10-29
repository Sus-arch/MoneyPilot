package storage

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

func NewPostgres(dsn string) *sql.DB {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("[Postgres] connection error: %v", err)
	}
	if err := db.Ping(); err != nil {
		log.Fatalf("[Postgres] ping failed: %v", err)
	}
	log.Println("✅ Connected to Postgres")
	return db
}

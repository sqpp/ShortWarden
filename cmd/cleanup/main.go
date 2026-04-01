package main

import (
	"context"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"shortwarden/internal/store"
)

func main() {
	dsn := os.Getenv("SHORTWARDEN_POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("SHORTWARDEN_POSTGRES_DSN must not be empty")
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("db: %v", err)
	}
	defer db.Close()

	q := store.New(db)
	if err := q.CleanupExpiredLinks(context.Background()); err != nil {
		log.Fatalf("cleanup: %v", err)
	}
	log.Printf("cleanup ok")
}


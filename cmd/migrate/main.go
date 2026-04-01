package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	var dsn string
	var dir string
	flag.StringVar(&dsn, "dsn", os.Getenv("SHORTWARDEN_POSTGRES_DSN"), "Postgres DSN")
	flag.StringVar(&dir, "dir", "db/migrations", "Migrations directory")
	flag.Parse()

	if dsn == "" {
		log.Fatal("missing -dsn (or SHORTWARDEN_POSTGRES_DSN)")
	}

	m, err := migrate.New("file://"+dir, dsn)
	if err != nil {
		log.Fatalf("migrate init: %v", err)
	}

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			fmt.Println("no change")
			return
		}
		log.Fatalf("migrate up: %v", err)
	}
	fmt.Println("migrated")
}


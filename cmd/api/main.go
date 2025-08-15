package main

import (
	"log"
	"net/http"
	"os"

	"p2p-chess/internal/http"
	"p2p-chess/internal/store"
)

func main() {
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN not set")
	}
	if err := store.RunMigrations(dsn); err != nil {
		log.Fatal("Migration error: ", err)
	}
	// TODO: Init JWT keys, etc.

	router := http.NewRouter()
	log.Fatal(http.ListenAndServe(":8080", router))
}

package main

import (
	"log"
	"net/http"
	"os"

	"p2p-chess/internal/auth"
	apihttp "p2p-chess/internal/http"
	"p2p-chess/internal/store"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file: ", err)
	}

	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		log.Fatal("DB_DSN not set")
	}
	if err := store.RunMigrations(dsn); err != nil {
		log.Fatal("Migration error: ", err)
	}

	// Initialize auth module
	if err := auth.Init(); err != nil {
		log.Fatal("Auth initialization error: ", err)
	}

	router := apihttp.NewRouter()
	log.Println("Server starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", router))
}

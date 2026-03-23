package main

import (
	"log"

	"clinic-backend/internal/mail"
	"clinic-backend/internal/platform/db"
	"clinic-backend/internal/worker"
)

func main() {
	// Initialize PostgreSQL for cross-checking job realities
	database, err := db.NewPostgresDB("localhost", "5432", "postgres", "postgres", "clinic")
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to DB for Worker: %v", err)
	}

	mailer := mail.NewLocalConsoleMailer()
	
	processor := worker.NewProcessor(database, mailer)

	log.Println("[WORKER] Booting background consumer connected to Redis on localhost:6379...")
	if err := processor.Start("localhost:6379"); err != nil {
		log.Fatalf("[WORKER] Crash executing async server: %v", err)
	}
}

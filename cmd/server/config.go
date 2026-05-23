package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type config struct {
	databaseURL string
	listenAddr  string
}

func loadConfig() config {
	if err := godotenv.Load(); err != nil {
		log.Printf("note: .env not loaded (%v); using existing environment", err)
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	return config{
		databaseURL: os.Getenv("DATABASE_URL"),
		listenAddr:  addr,
	}
}

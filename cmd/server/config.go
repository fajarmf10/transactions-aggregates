package main

import "os"

type config struct {
	databaseURL string
	listenAddr  string
}

func loadConfig() config {
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	return config{
		databaseURL: os.Getenv("DATABASE_URL"),
		listenAddr:  addr,
	}
}

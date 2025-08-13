package main

import (
	"backend/internal/config"
	httpapi "backend/internal/http"
	"log"
	"net/http"
	"time"

	"github.com/joho/godotenv"
)

func main() {

	// --------------------- LOAD CONFIG --------------------------
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, continuing with defaults...")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// --------------------- SET UP SERVER --------------------------

	api := httpapi.New(cfg)

	srv := &http.Server{
		Addr:              cfg.App.Addr,
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", cfg.App.Addr)
	log.Fatal(srv.ListenAndServe())

}

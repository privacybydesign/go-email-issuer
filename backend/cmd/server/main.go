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

	cfg := config.Load()

	// --------------------- SET UP SERVER --------------------------

	handler := httpapi.Routes()

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", cfg.Addr)
	log.Fatal(srv.ListenAndServe())

}

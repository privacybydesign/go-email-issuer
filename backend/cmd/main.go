package main

import (
	"backend/internal/config"
	api "backend/internal/http"
	"flag"
	"log"
)

func main() {

	// --------------------- LOAD CONFIG --------------------------

	cfgPath := flag.String("config", "config.json", "Path to the config file")
	flag.Parse()

	if *cfgPath == "" {
		log.Fatal("Please provide a config file path using the -config flag")
	}

	log.Printf("Loading configuration from %s", *cfgPath)

	cfg, err := config.LoadFromFile(*cfgPath)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// --------------------- SET UP SERVER --------------------------
	serv := api.NewServer(cfg)

	log.Printf("listening on %s", cfg.App.Addr)
	log.Fatal(serv.ListenAndServe())

}

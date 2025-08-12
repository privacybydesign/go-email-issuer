package config

import "os"

type Config struct {
	Addr     string
	BASE_URL string
	SECRET   string
	TTLMIN   string
}

// func to load env vars, accompanied by defaults

func Load() Config {
	return Config{
		Addr:     getenv("ADDR", ":8080"),
		BASE_URL: getenv("BASE_URL", "localhost"),
		SECRET:   getenv("SECRET", "changeme"),
		TTLMIN:   getenv("TTLMIN", "15"),
	}
}

func getenv(key, def string) string {

	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

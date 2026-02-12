package config

import (
	"log"
	"os"
)

type Config struct {
	DatabaseURL string
	Port        string
}

func Load() Config {
	return Config{
		DatabaseURL: mustEnv("DATABASE_URL"),
		Port:        envOr("PORT", "8080"),
	}
}

func mustEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("missing env: %s", key)
	}
	return value
}

func envOr(key, def string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return def
}

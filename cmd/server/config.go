package main

import "os"

type Config struct {
	Port   string
	Domain string
	DBPath string
	Secret string
}

func LoadConfig() Config {
	return Config{
		Port:   getenv("PORT", "8080"),
		Domain: getenv("DOMAIN", "localhost"),
		DBPath: getenv("DATABASE_PATH", "./gofolio.db"),
		Secret: os.Getenv("AUTH_SECRET"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

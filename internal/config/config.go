package config

import (
	"fmt"
	"os"
)

type Config struct {
	HTTPAddr string
	DB       DBConfig
}

type DBConfig struct {
	DSN string
}

func Load() (*Config, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		HTTPAddr: ":" + port,
		DB:       DBConfig{DSN: dsn},
	}, nil
}

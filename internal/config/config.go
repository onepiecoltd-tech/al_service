package config

import (
	"fmt"
	"os"
)

type Config struct {
	Env      string
	HTTPAddr  string
	DB        DBConfig
	JWTSecret string
}

type DBConfig struct {
	DSN string
}

func Load() (*Config, error) {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	return &Config{
		Env:      env,
		HTTPAddr:  ":" + port,
		DB:        DBConfig{DSN: dsn},
		JWTSecret: jwtSecret,
	}, nil
}

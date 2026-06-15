package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/craftbyte/learning_languages/services/internal/config"
	"github.com/craftbyte/learning_languages/services/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	srv := server.New(cfg)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

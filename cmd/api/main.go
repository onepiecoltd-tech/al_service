package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/craftbyte/learning_languages/services/internal/config"
	"github.com/craftbyte/learning_languages/services/internal/server"
	"github.com/craftbyte/learning_languages/services/pkg/logger"
)

//	@title			Learning Languages API
//	@version		0.1.0
//	@description	REST API backend for craftbyte/learning_languages.
//	@host			localhost:8080
//	@BasePath		/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//	@description				Type "Bearer <token>"

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger.Init(cfg.Env)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := pgxpool.New(ctx, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("connect db: %v", err)
	}
	defer db.Close()

	srv := server.New(cfg, db)
	if err := srv.Run(ctx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

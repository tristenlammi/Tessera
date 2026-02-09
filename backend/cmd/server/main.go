package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/tessera/tessera/internal/config"
	"github.com/tessera/tessera/internal/database"
	"github.com/tessera/tessera/internal/logger"
	"github.com/tessera/tessera/internal/server"
	"github.com/tessera/tessera/internal/storage"
)

func main() {
	// Initialize logger
	log := logger.New()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().
		Str("env", cfg.App.Env).
		Str("host", cfg.Server.Host).
		Int("port", cfg.Server.Port).
		Msg("Starting Tessera")

	// Run database migrations before connecting the pool
	migrationsPath := "file:///app/migrations"
	if p := os.Getenv("MIGRATIONS_PATH"); p != "" {
		migrationsPath = p
	}
	if err := database.RunMigrations(cfg.Database, migrationsPath, log); err != nil {
		log.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	// Connect to database
	db, err := database.Connect(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("Connected to PostgreSQL")

	// Connect to Redis
	rdb, err := database.ConnectRedis(cfg.Redis)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	defer rdb.Close()
	log.Info().Msg("Connected to Redis")

	// Initialize MinIO storage
	store, err := storage.NewMinIO(cfg.Storage)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize MinIO storage")
	}
	log.Info().Msg("Connected to MinIO")

	// Create and start server
	srv := server.New(cfg, db, rdb, store, log)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	<-quit
	log.Info().Msg("Shutting down server...")

	if err := srv.Shutdown(); err != nil {
		log.Error().Err(err).Msg("Server shutdown error")
	}

	log.Info().Msg("Server stopped")
}

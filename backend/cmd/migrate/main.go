package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"github.com/tessera/tessera/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Build database URL for golang-migrate (uses pgx5 scheme)
	dbURL := fmt.Sprintf(
		"pgx5://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	migrationsPath := "file:///app/migrations"
	// Allow override via env var (useful for local dev)
	if p := os.Getenv("MIGRATIONS_PATH"); p != "" {
		migrationsPath = p
	}

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	if len(flag.Args()) == 0 && len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		fmt.Println("Migrations applied successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		fmt.Println("All migrations reverted")

	case "steps":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate steps <n> (positive=up, negative=down)")
		}
		n, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid step count: %v", err)
		}
		if err := m.Steps(n); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration steps failed: %v", err)
		}
		fmt.Printf("Applied %d migration steps\n", n)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get version: %v", err)
		}
		fmt.Printf("Version: %d, Dirty: %v\n", version, dirty)

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate force <version>")
		}
		v, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid version: %v", err)
		}
		if err := m.Force(v); err != nil {
			log.Fatalf("Force version failed: %v", err)
		}
		fmt.Printf("Forced version to %d\n", v)

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command> [args]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  up        Apply all pending migrations")
	fmt.Println("  down      Revert all migrations")
	fmt.Println("  steps <n> Apply n migrations (positive=up, negative=down)")
	fmt.Println("  version   Print current migration version")
	fmt.Println("  force <v> Force set migration version (no migrations run)")
}

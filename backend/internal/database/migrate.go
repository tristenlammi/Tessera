package database

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/rs/zerolog"
	"github.com/tessera/tessera/internal/config"
)

// RunMigrations applies all pending database migrations.
// It uses golang-migrate with the pgx5 driver.
func RunMigrations(cfg config.DatabaseConfig, migrationsPath string, log zerolog.Logger) error {
	dbURL := fmt.Sprintf(
		"pgx5://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.Name,
		cfg.SSLMode,
	)

	m, err := migrate.New(migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}
	defer m.Close()

	version, dirty, _ := m.Version()
	log.Info().Uint("current_version", version).Bool("dirty", dirty).Msg("Current migration version")

	if err := m.Up(); err != nil {
		if err == migrate.ErrNoChange {
			log.Info().Msg("Database schema is up to date")
			return nil
		}
		return fmt.Errorf("migration failed: %w", err)
	}

	newVersion, _, _ := m.Version()
	log.Info().Uint("new_version", newVersion).Msg("Migrations applied successfully")
	return nil
}

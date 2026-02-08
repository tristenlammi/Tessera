package repository

import (
	"context"
	"encoding/json"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SettingsRepository handles settings persistence
type SettingsRepository struct {
	db *pgxpool.Pool
}

// NewSettingsRepository creates a new settings repository
func NewSettingsRepository(db *pgxpool.Pool) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get retrieves a setting by key
func (r *SettingsRepository) Get(ctx context.Context, key string) (map[string]interface{}, error) {
	query := `SELECT value FROM settings WHERE key = $1`

	var valueJSON []byte
	err := r.db.QueryRow(ctx, query, key).Scan(&valueJSON)
	if err != nil {
		return nil, err
	}

	var value map[string]interface{}
	if err := json.Unmarshal(valueJSON, &value); err != nil {
		return nil, err
	}

	return value, nil
}

// Set saves a setting
func (r *SettingsRepository) Set(ctx context.Context, key string, value interface{}) error {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return err
	}

	query := `
		INSERT INTO settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()
	`

	_, err = r.db.Exec(ctx, query, key, valueJSON)
	return err
}

// GetModules retrieves module settings
func (r *SettingsRepository) GetModules(ctx context.Context) (map[string]bool, error) {
	value, err := r.Get(ctx, "modules")
	if err != nil {
		// Return defaults if not found
		return map[string]bool{
			"documents": false,
			"pdf":       false,
			"tasks":     false,
			"calendar":  false,
			"contacts":  false,
			"email":     false,
		}, nil
	}

	modules := make(map[string]bool)
	for k, v := range value {
		if b, ok := v.(bool); ok {
			modules[k] = b
		}
	}

	return modules, nil
}

// SetModules saves module settings
func (r *SettingsRepository) SetModules(ctx context.Context, modules map[string]bool) error {
	return r.Set(ctx, "modules", modules)
}

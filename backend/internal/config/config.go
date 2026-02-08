package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all application configuration
type Config struct {
	App        AppConfig
	Server     ServerConfig
	Database   DatabaseConfig
	Redis      RedisConfig
	Storage    StorageConfig
	JWT        JWTConfig
	Upload     UploadConfig
	Encryption EncryptionConfig
}

type AppConfig struct {
	Env   string
	Debug bool
}

type ServerConfig struct {
	Host        string
	Port        int
	FrontendURL string
}

type DatabaseConfig struct {
	Host     string
	Port     int
	Name     string
	User     string
	Password string
	SSLMode  string
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type JWTConfig struct {
	Secret        string
	Expiry        time.Duration
	RefreshExpiry time.Duration
}

type UploadConfig struct {
	MaxSize   int64
	ChunkSize int64
}

type EncryptionConfig struct {
	MasterKey string // Base64 encoded 32-byte key for AES-256
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (ignore errors in production)
	_ = godotenv.Load()

	cfg := &Config{
		App: AppConfig{
			Env:   getEnv("APP_ENV", "development"),
			Debug: getEnvBool("APP_DEBUG", true),
		},
		Server: ServerConfig{
			Host:        getEnv("SERVER_HOST", "0.0.0.0"),
			Port:        getEnvInt("SERVER_PORT", 8080),
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvInt("DB_PORT", 5432),
			Name:     getEnv("DB_NAME", "tessera"),
			User:     getEnv("DB_USER", "tessera"),
			Password: getEnv("DB_PASSWORD", ""),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		Storage: StorageConfig{
			Endpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
			AccessKey: getEnv("MINIO_ACCESS_KEY", ""),
			SecretKey: getEnv("MINIO_SECRET_KEY", ""),
			Bucket:    getEnv("MINIO_BUCKET", "tessera-files"),
			UseSSL:    getEnvBool("MINIO_USE_SSL", false),
		},
		JWT: JWTConfig{
			Secret:        getEnv("JWT_SECRET", "change-me-in-production"),
			Expiry:        getEnvDuration("JWT_EXPIRY", 15*time.Minute),
			RefreshExpiry: getEnvDuration("JWT_REFRESH_EXPIRY", 7*24*time.Hour),
		},
		Upload: UploadConfig{
			MaxSize:   getEnvInt64("MAX_UPLOAD_SIZE", 10*1024*1024*1024), // 10GB
			ChunkSize: getEnvInt64("CHUNK_SIZE", 10*1024*1024),           // 10MB
		},
		Encryption: EncryptionConfig{
			// ENCRYPTION_KEY should be a base64-encoded 32-byte key
			// Generate with: openssl rand -base64 32
			MasterKey: getEnv("ENCRYPTION_KEY", ""),
		},
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if boolVal, err := strconv.ParseBool(value); err == nil {
			return boolVal
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if value, ok := os.LookupEnv(key); ok {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return fallback
}

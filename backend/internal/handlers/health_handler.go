package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

var startTime = time.Now()

// HealthHandler handles health check endpoints
type HealthHandler struct {
	log   zerolog.Logger
	db    *pgxpool.Pool
	rdb   *redis.Client
	minio *minio.Client
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(log zerolog.Logger, db *pgxpool.Pool, rdb *redis.Client, minio *minio.Client) *HealthHandler {
	return &HealthHandler{
		log:   log,
		db:    db,
		rdb:   rdb,
		minio: minio,
	}
}

// ComponentHealth represents the health status of a component
type ComponentHealth struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents the overall health response
type HealthResponse struct {
	Status     string                     `json:"status"`
	Timestamp  string                     `json:"timestamp"`
	Version    string                     `json:"version"`
	Uptime     string                     `json:"uptime"`
	Components map[string]ComponentHealth `json:"components,omitempty"`
}

// Liveness returns a simple liveness check (is the app running?)
// GET /health
func (h *HealthHandler) Liveness(c *fiber.Ctx) error {
	return c.JSON(HealthResponse{
		Status:    "ok",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   "1.0.0",
		Uptime:    time.Since(startTime).Round(time.Second).String(),
	})
}

// Readiness returns a comprehensive readiness check (is the app ready to serve?)
// GET /ready
func (h *HealthHandler) Readiness(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.Context(), 5*time.Second)
	defer cancel()

	components := make(map[string]ComponentHealth)
	overallStatus := "ok"

	// Check PostgreSQL
	dbHealth := h.checkPostgres(ctx)
	components["postgres"] = dbHealth
	if dbHealth.Status != "ok" {
		overallStatus = "degraded"
	}

	// Check Redis
	redisHealth := h.checkRedis(ctx)
	components["redis"] = redisHealth
	if redisHealth.Status != "ok" {
		overallStatus = "degraded"
	}

	// Check MinIO
	minioHealth := h.checkMinIO(ctx)
	components["minio"] = minioHealth
	if minioHealth.Status != "ok" {
		overallStatus = "degraded"
	}

	response := HealthResponse{
		Status:     overallStatus,
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		Version:    "1.0.0",
		Uptime:     time.Since(startTime).Round(time.Second).String(),
		Components: components,
	}

	if overallStatus != "ok" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(response)
	}

	return c.JSON(response)
}

// checkPostgres verifies PostgreSQL connectivity
func (h *HealthHandler) checkPostgres(ctx context.Context) ComponentHealth {
	start := time.Now()

	if h.db == nil {
		return ComponentHealth{
			Status:  "error",
			Message: "Database pool is nil",
		}
	}

	var result int
	err := h.db.QueryRow(ctx, "SELECT 1").Scan(&result)
	latency := time.Since(start)

	if err != nil {
		h.log.Error().Err(err).Msg("PostgreSQL health check failed")
		return ComponentHealth{
			Status:  "error",
			Latency: latency.String(),
			Message: "Query failed",
		}
	}

	stats := h.db.Stat()
	return ComponentHealth{
		Status:  "ok",
		Latency: latency.String(),
		Message: fmt.Sprintf("conns: total=%d idle=%d acquired=%d", stats.TotalConns(), stats.IdleConns(), stats.AcquiredConns()),
	}
}

// checkRedis verifies Redis connectivity
func (h *HealthHandler) checkRedis(ctx context.Context) ComponentHealth {
	start := time.Now()

	if h.rdb == nil {
		return ComponentHealth{
			Status:  "error",
			Message: "Redis client is nil",
		}
	}

	_, err := h.rdb.Ping(ctx).Result()
	latency := time.Since(start)

	if err != nil {
		h.log.Error().Err(err).Msg("Redis health check failed")
		return ComponentHealth{
			Status:  "error",
			Latency: latency.String(),
			Message: "Ping failed",
		}
	}

	return ComponentHealth{
		Status:  "ok",
		Latency: latency.String(),
	}
}

// checkMinIO verifies MinIO connectivity
func (h *HealthHandler) checkMinIO(ctx context.Context) ComponentHealth {
	start := time.Now()

	if h.minio == nil {
		return ComponentHealth{
			Status:  "error",
			Message: "MinIO client is nil",
		}
	}

	_, err := h.minio.ListBuckets(ctx)
	latency := time.Since(start)

	if err != nil {
		h.log.Error().Err(err).Msg("MinIO health check failed")
		return ComponentHealth{
			Status:  "error",
			Latency: latency.String(),
			Message: "ListBuckets failed",
		}
	}

	return ComponentHealth{
		Status:  "ok",
		Latency: latency.String(),
	}
}

package middleware

import (
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/tessera/tessera/internal/metrics"
)

// Metrics returns a middleware that collects Prometheus metrics
func Metrics() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Skip metrics endpoint to avoid recursion
		if c.Path() == "/metrics" {
			return c.Next()
		}

		start := time.Now()

		// Track active connections
		metrics.ActiveConnections.Inc()
		defer metrics.ActiveConnections.Dec()

		// Process request
		err := c.Next()

		// Calculate metrics after response
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Response().StatusCode())
		method := c.Method()
		path := metrics.NormalizePath(c.Path())

		// Record metrics
		metrics.HTTPRequestsTotal.WithLabelValues(method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(method, path).Observe(duration)

		// Request size (approximate from Content-Length header)
		if reqSize := c.Request().Header.ContentLength(); reqSize > 0 {
			metrics.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(reqSize))
		}

		// Response size
		respSize := len(c.Response().Body())
		if respSize > 0 {
			metrics.HTTPResponseSize.WithLabelValues(method, path).Observe(float64(respSize))
		}

		// Track errors
		statusCode := c.Response().StatusCode()
		if statusCode >= 400 {
			errorCode := strconv.Itoa(statusCode)
			metrics.ErrorsTotal.WithLabelValues("http", errorCode).Inc()
		}

		return err
	}
}

// RecordAuthAttempt records an authentication attempt
func RecordAuthAttempt(authType, status string) {
	metrics.AuthAttempts.WithLabelValues(authType, status).Inc()
}

// RecordFileUpload records a file upload
func RecordFileUpload(status string, bytes int64) {
	metrics.FileUploads.WithLabelValues(status).Inc()
	if status == "success" && bytes > 0 {
		metrics.FileUploadBytes.Add(float64(bytes))
	}
}

// RecordFileDownload records a file download
func RecordFileDownload(status string, bytes int64) {
	metrics.FileDownloads.WithLabelValues(status).Inc()
	if status == "success" && bytes > 0 {
		metrics.FileDownloadBytes.Add(float64(bytes))
	}
}

// RecordEmailSync records an email sync operation
func RecordEmailSync(status string) {
	metrics.EmailsSynced.WithLabelValues(status).Inc()
}

// RecordEmailSent records an email being sent
func RecordEmailSent(status string) {
	metrics.EmailsSent.WithLabelValues(status).Inc()
}

// RecordJobProcessed records a background job completion
func RecordJobProcessed(jobType, status string, duration time.Duration) {
	metrics.JobsProcessed.WithLabelValues(jobType, status).Inc()
	metrics.JobDuration.WithLabelValues(jobType).Observe(duration.Seconds())
}

// RecordCacheHit records a cache hit
func RecordCacheHit(cache string) {
	metrics.CacheHits.WithLabelValues(cache).Inc()
}

// RecordCacheMiss records a cache miss
func RecordCacheMiss(cache string) {
	metrics.CacheMisses.WithLabelValues(cache).Inc()
}

// UpdateWebSocketConnections updates the WebSocket connection gauge
func UpdateWebSocketConnections(delta int) {
	if delta > 0 {
		metrics.WebSocketConnections.Add(float64(delta))
	} else {
		metrics.WebSocketConnections.Sub(float64(-delta))
	}
}

// RecordWebSocketMessage records a WebSocket message
func RecordWebSocketMessage(direction string) {
	metrics.WebSocketMessages.WithLabelValues(direction).Inc()
}

// UpdateJobsQueued updates the jobs queued gauge
func UpdateJobsQueued(count int) {
	metrics.JobsQueued.Set(float64(count))
}

// UpdateStorageMetrics updates storage-related metrics
func UpdateStorageMetrics(filesBytes, attachmentsBytes, thumbnailsBytes int64, totalFiles int) {
	metrics.StorageUsedBytes.WithLabelValues("files").Set(float64(filesBytes))
	metrics.StorageUsedBytes.WithLabelValues("attachments").Set(float64(attachmentsBytes))
	metrics.StorageUsedBytes.WithLabelValues("thumbnails").Set(float64(thumbnailsBytes))
	metrics.StorageTotalFiles.Set(float64(totalFiles))
}

// UpdateUserMetrics updates user-related metrics
func UpdateUserMetrics(registered, active24h int) {
	metrics.RegisteredUsers.Set(float64(registered))
	metrics.ActiveUsers.Set(float64(active24h))
}

// UpdateDBConnections updates database connection metrics
func UpdateDBConnections(total, idle, active int) {
	metrics.DBConnections.WithLabelValues("total").Set(float64(total))
	metrics.DBConnections.WithLabelValues("idle").Set(float64(idle))
	metrics.DBConnections.WithLabelValues("active").Set(float64(active))
}

// UpdateActiveSessions updates the active sessions gauge
func UpdateActiveSessions(count int) {
	metrics.ActiveSessions.Set(float64(count))
}

// UpdateEmailAccounts updates the email accounts gauge
func UpdateEmailAccounts(count int) {
	metrics.EmailAccounts.Set(float64(count))
}

// RecordDBQuery records a database query duration
func RecordDBQuery(operation string, duration time.Duration) {
	metrics.DBQueryDuration.WithLabelValues(operation).Observe(duration.Seconds())
}

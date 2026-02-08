package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP Metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tessera_http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"method", "path"},
	)

	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tessera_http_request_size_bytes",
			Help:    "Size of HTTP requests in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100B to 10GB
		},
		[]string{"method", "path"},
	)

	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tessera_http_response_size_bytes",
			Help:    "Size of HTTP responses in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8), // 100B to 10GB
		},
		[]string{"method", "path"},
	)

	// Active connections
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// WebSocket Metrics
	WebSocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_websocket_connections",
			Help: "Number of active WebSocket connections",
		},
	)

	WebSocketMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_websocket_messages_total",
			Help: "Total number of WebSocket messages",
		},
		[]string{"direction"}, // "inbound" or "outbound"
	)

	// Authentication Metrics
	AuthAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_auth_attempts_total",
			Help: "Total number of authentication attempts",
		},
		[]string{"type", "status"}, // type: "login", "refresh", "2fa"; status: "success", "failure"
	)

	ActiveSessions = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_active_sessions",
			Help: "Number of active user sessions",
		},
	)

	// File Operations
	FileUploads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_file_uploads_total",
			Help: "Total number of file uploads",
		},
		[]string{"status"}, // "success", "failure"
	)

	FileDownloads = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_file_downloads_total",
			Help: "Total number of file downloads",
		},
		[]string{"status"},
	)

	FileUploadBytes = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tessera_file_upload_bytes_total",
			Help: "Total bytes uploaded",
		},
	)

	FileDownloadBytes = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "tessera_file_download_bytes_total",
			Help: "Total bytes downloaded",
		},
	)

	// Storage Metrics
	StorageUsedBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tessera_storage_used_bytes",
			Help: "Storage used in bytes",
		},
		[]string{"type"}, // "files", "attachments", "thumbnails"
	)

	StorageTotalFiles = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_storage_total_files",
			Help: "Total number of files stored",
		},
	)

	// Email Metrics
	EmailsSynced = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_emails_synced_total",
			Help: "Total number of emails synced",
		},
		[]string{"status"},
	)

	EmailsSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_emails_sent_total",
			Help: "Total number of emails sent",
		},
		[]string{"status"},
	)

	EmailAccounts = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_email_accounts",
			Help: "Number of connected email accounts",
		},
	)

	// Background Jobs
	JobsProcessed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_jobs_processed_total",
			Help: "Total number of background jobs processed",
		},
		[]string{"type", "status"}, // type: "thumbnail", "cleanup", etc; status: "success", "failure"
	)

	JobsQueued = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_jobs_queued",
			Help: "Number of jobs currently in queue",
		},
	)

	JobDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tessera_job_duration_seconds",
			Help:    "Duration of background jobs in seconds",
			Buckets: []float64{.1, .5, 1, 2.5, 5, 10, 30, 60, 120},
		},
		[]string{"type"},
	)

	// Database Metrics
	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "tessera_db_query_duration_seconds",
			Help:    "Duration of database queries in seconds",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"operation"}, // "select", "insert", "update", "delete"
	)

	DBConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tessera_db_connections",
			Help: "Number of database connections",
		},
		[]string{"state"}, // "active", "idle", "total"
	)

	// User Metrics
	RegisteredUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_registered_users",
			Help: "Total number of registered users",
		},
	)

	ActiveUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "tessera_active_users_24h",
			Help: "Number of users active in the last 24 hours",
		},
	)

	// Error Metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_errors_total",
			Help: "Total number of errors",
		},
		[]string{"type", "code"}, // type: "http", "db", "storage"; code: error code
	)

	// Cache Metrics
	CacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache"}, // "redis", "memory"
	)

	CacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tessera_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache"},
	)
)

// NormalizePath normalizes a path for metrics labels to avoid high cardinality
func NormalizePath(path string) string {
	// Replace UUIDs with placeholder
	// This prevents high cardinality from unique IDs
	normalized := path

	// Common patterns to normalize
	patterns := map[string]string{
		"/api/files/":     "/api/files/:id",
		"/api/folders/":   "/api/folders/:id",
		"/api/share/":     "/api/share/:id",
		"/api/users/":     "/api/users/:id",
		"/api/emails/":    "/api/emails/:id",
		"/api/contacts/":  "/api/contacts/:id",
		"/api/tasks/":     "/api/tasks/:id",
		"/api/events/":    "/api/events/:id",
		"/api/documents/": "/api/documents/:id",
	}

	for prefix, replacement := range patterns {
		if len(path) > len(prefix) && path[:len(prefix)] == prefix {
			return replacement
		}
	}

	return normalized
}

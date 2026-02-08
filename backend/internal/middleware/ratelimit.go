package middleware

import (
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// RateLimiter implements a token bucket rate limiter
type RateLimiter struct {
	buckets      map[string]*bucket
	mu           sync.RWMutex
	rate         int           // tokens per interval
	interval     time.Duration // refill interval
	maxBurst     int           // maximum tokens
	cleanupEvery time.Duration
	stopChan     chan struct{} // channel to stop cleanup goroutine
}

type bucket struct {
	tokens    int
	lastRefil time.Time
}

// RateLimitConfig holds rate limiter configuration
type RateLimitConfig struct {
	// Rate is the number of requests allowed per interval
	Rate int
	// Interval is the time period for the rate (e.g., time.Minute)
	Interval time.Duration
	// MaxBurst is the maximum number of requests allowed in a burst
	MaxBurst int
	// KeyGenerator generates a key for rate limiting (default: IP address)
	KeyGenerator func(*fiber.Ctx) string
	// SkipPaths are paths that bypass rate limiting
	SkipPaths []string
	// OnLimitReached is called when rate limit is exceeded
	OnLimitReached func(*fiber.Ctx) error
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Rate:     100,
		Interval: time.Minute,
		MaxBurst: 10,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		SkipPaths: []string{"/health", "/metrics"},
		OnLimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
				},
			})
		},
	}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config RateLimitConfig) *RateLimiter {
	if config.Rate <= 0 {
		config.Rate = 100
	}
	if config.Interval <= 0 {
		config.Interval = time.Minute
	}
	if config.MaxBurst <= 0 {
		config.MaxBurst = config.Rate / 10
		if config.MaxBurst < 1 {
			config.MaxBurst = 1
		}
	}

	rl := &RateLimiter{
		buckets:      make(map[string]*bucket),
		rate:         config.Rate,
		interval:     config.Interval,
		maxBurst:     config.MaxBurst,
		cleanupEvery: 5 * time.Minute,
		stopChan:     make(chan struct{}),
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.buckets[key]

	if !exists {
		rl.buckets[key] = &bucket{
			tokens:    rl.maxBurst - 1,
			lastRefil: now,
		}
		return true
	}

	// Refill tokens based on time elapsed
	elapsed := now.Sub(b.lastRefil)
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rate) / rl.interval.Seconds())
	if tokensToAdd > 0 {
		b.tokens += tokensToAdd
		if b.tokens > rl.maxBurst {
			b.tokens = rl.maxBurst
		}
		b.lastRefil = now
	}

	// Check if we have tokens
	if b.tokens > 0 {
		b.tokens--
		return true
	}

	return false
}

// cleanup removes stale buckets periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			threshold := time.Now().Add(-rl.cleanupEvery)
			for key, b := range rl.buckets {
				if b.lastRefil.Before(threshold) {
					delete(rl.buckets, key)
				}
			}
			rl.mu.Unlock()
		case <-rl.stopChan:
			return
		}
	}
}

// Stop stops the rate limiter cleanup goroutine
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// RateLimiterRegistry tracks all rate limiters for cleanup on shutdown
var (
	rateLimiterRegistry   []*RateLimiter
	rateLimiterRegistryMu sync.Mutex
)

// StopAllRateLimiters stops all registered rate limiters (call on shutdown)
func StopAllRateLimiters() {
	rateLimiterRegistryMu.Lock()
	defer rateLimiterRegistryMu.Unlock()
	for _, rl := range rateLimiterRegistry {
		rl.Stop()
	}
	rateLimiterRegistry = nil
}

// RateLimitMiddleware creates a Fiber middleware for rate limiting
func RateLimitMiddleware(config RateLimitConfig) fiber.Handler {
	if config.KeyGenerator == nil {
		config.KeyGenerator = func(c *fiber.Ctx) string {
			return c.IP()
		}
	}
	if config.OnLimitReached == nil {
		config.OnLimitReached = func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Rate limit exceeded. Please try again later.",
				},
			})
		}
	}

	rl := NewRateLimiter(config)

	// Register for cleanup on shutdown
	rateLimiterRegistryMu.Lock()
	rateLimiterRegistry = append(rateLimiterRegistry, rl)
	rateLimiterRegistryMu.Unlock()

	return func(c *fiber.Ctx) error {
		// Check if path should be skipped
		path := c.Path()
		for _, skipPath := range config.SkipPaths {
			if path == skipPath {
				return c.Next()
			}
		}

		key := config.KeyGenerator(c)
		if !rl.Allow(key) {
			return config.OnLimitReached(c)
		}

		return c.Next()
	}
}

// StrictRateLimitConfig returns a stricter configuration for sensitive endpoints
func StrictRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Rate:     10,
		Interval: time.Minute,
		MaxBurst: 5,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		OnLimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many attempts. Please wait before trying again.",
				},
			})
		},
	}
}

// LoginRateLimitConfig returns configuration for login endpoint
func LoginRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Rate:     5,
		Interval: time.Minute,
		MaxBurst: 3,
		KeyGenerator: func(c *fiber.Ctx) string {
			// Rate limit by IP + attempted email
			return c.IP()
		},
		OnLimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"success": false,
				"error": fiber.Map{
					"code":    "RATE_LIMIT_EXCEEDED",
					"message": "Too many login attempts. Please try again in a few minutes.",
				},
			})
		},
	}
}

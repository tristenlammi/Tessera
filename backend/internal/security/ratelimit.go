package security

import (
	"context"
	"sync"
	"time"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	requests map[string]*rateLimitEntry
	mu       sync.RWMutex
	limit    int
	window   time.Duration
}

type rateLimitEntry struct {
	count      int
	firstSeen  time.Time
	blocked    bool
	blockUntil time.Time
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string]*rateLimitEntry),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	entry, exists := rl.requests[key]
	if !exists {
		rl.requests[key] = &rateLimitEntry{
			count:     1,
			firstSeen: now,
		}
		return true
	}

	// Check if blocked
	if entry.blocked {
		if now.Before(entry.blockUntil) {
			return false
		}
		// Unblock and reset
		entry.blocked = false
		entry.count = 1
		entry.firstSeen = now
		return true
	}

	// Check if window has passed
	if now.Sub(entry.firstSeen) > rl.window {
		entry.count = 1
		entry.firstSeen = now
		return true
	}

	// Increment count
	entry.count++

	// Check if over limit
	if entry.count > rl.limit {
		return false
	}

	return true
}

// Block temporarily blocks a key
func (rl *RateLimiter) Block(key string, duration time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.requests[key]
	if !exists {
		entry = &rateLimitEntry{}
		rl.requests[key] = entry
	}

	entry.blocked = true
	entry.blockUntil = time.Now().Add(duration)
}

// IsBlocked checks if a key is blocked
func (rl *RateLimiter) IsBlocked(key string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.requests[key]
	if !exists {
		return false
	}

	if entry.blocked && time.Now().Before(entry.blockUntil) {
		return true
	}

	return false
}

// GetRemaining returns the remaining requests for a key
func (rl *RateLimiter) GetRemaining(key string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	entry, exists := rl.requests[key]
	if !exists {
		return rl.limit
	}

	// Check if window has passed
	if time.Now().Sub(entry.firstSeen) > rl.window {
		return rl.limit
	}

	remaining := rl.limit - entry.count
	if remaining < 0 {
		return 0
	}
	return remaining
}

// cleanup removes old entries periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, entry := range rl.requests {
			if now.Sub(entry.firstSeen) > rl.window*2 && (!entry.blocked || now.After(entry.blockUntil)) {
				delete(rl.requests, key)
			}
		}
		rl.mu.Unlock()
	}
}

// LoginAttemptLimiter specifically handles login attempts with progressive blocking
type LoginAttemptLimiter struct {
	attempts map[string]*loginAttempts
	mu       sync.RWMutex
}

type loginAttempts struct {
	failedCount  int
	lastAttempt  time.Time
	blockedUntil time.Time
}

// NewLoginAttemptLimiter creates a new login attempt limiter
func NewLoginAttemptLimiter() *LoginAttemptLimiter {
	lal := &LoginAttemptLimiter{
		attempts: make(map[string]*loginAttempts),
	}

	// Start cleanup goroutine
	go lal.cleanup()

	return lal
}

// RecordFailedAttempt records a failed login attempt
func (lal *LoginAttemptLimiter) RecordFailedAttempt(key string) {
	lal.mu.Lock()
	defer lal.mu.Unlock()

	now := time.Now()

	attempt, exists := lal.attempts[key]
	if !exists {
		attempt = &loginAttempts{}
		lal.attempts[key] = attempt
	}

	// Reset if last attempt was more than 24 hours ago
	if now.Sub(attempt.lastAttempt) > 24*time.Hour {
		attempt.failedCount = 0
	}

	attempt.failedCount++
	attempt.lastAttempt = now

	// Progressive blocking
	switch {
	case attempt.failedCount >= 10:
		attempt.blockedUntil = now.Add(24 * time.Hour)
	case attempt.failedCount >= 5:
		attempt.blockedUntil = now.Add(1 * time.Hour)
	case attempt.failedCount >= 3:
		attempt.blockedUntil = now.Add(5 * time.Minute)
	}
}

// RecordSuccessfulAttempt resets the failed count on successful login
func (lal *LoginAttemptLimiter) RecordSuccessfulAttempt(key string) {
	lal.mu.Lock()
	defer lal.mu.Unlock()

	delete(lal.attempts, key)
}

// IsBlocked checks if a key is blocked
func (lal *LoginAttemptLimiter) IsBlocked(key string) (bool, time.Duration) {
	lal.mu.RLock()
	defer lal.mu.RUnlock()

	attempt, exists := lal.attempts[key]
	if !exists {
		return false, 0
	}

	now := time.Now()
	if now.Before(attempt.blockedUntil) {
		return true, attempt.blockedUntil.Sub(now)
	}

	return false, 0
}

// GetFailedCount returns the number of failed attempts
func (lal *LoginAttemptLimiter) GetFailedCount(key string) int {
	lal.mu.RLock()
	defer lal.mu.RUnlock()

	attempt, exists := lal.attempts[key]
	if !exists {
		return 0
	}
	return attempt.failedCount
}

func (lal *LoginAttemptLimiter) cleanup() {
	ticker := time.NewTicker(time.Hour)
	for range ticker.C {
		lal.mu.Lock()
		now := time.Now()
		for key, attempt := range lal.attempts {
			if now.Sub(attempt.lastAttempt) > 24*time.Hour {
				delete(lal.attempts, key)
			}
		}
		lal.mu.Unlock()
	}
}

// CSRFToken generates and validates CSRF tokens
type CSRFTokenManager struct {
	tokens map[string]time.Time
	mu     sync.RWMutex
	ttl    time.Duration
}

// NewCSRFTokenManager creates a new CSRF token manager
func NewCSRFTokenManager(ttl time.Duration) *CSRFTokenManager {
	ctm := &CSRFTokenManager{
		tokens: make(map[string]time.Time),
		ttl:    ttl,
	}

	// Start cleanup goroutine
	go ctm.cleanup()

	return ctm
}

// Generate creates a new CSRF token
func (ctm *CSRFTokenManager) Generate(ctx context.Context) (string, error) {
	token, err := GenerateSecureToken(32)
	if err != nil {
		return "", err
	}

	ctm.mu.Lock()
	ctm.tokens[token] = time.Now()
	ctm.mu.Unlock()

	return token, nil
}

// Validate checks if a CSRF token is valid
func (ctm *CSRFTokenManager) Validate(token string) bool {
	ctm.mu.Lock()
	defer ctm.mu.Unlock()

	created, exists := ctm.tokens[token]
	if !exists {
		return false
	}

	// Check if expired
	if time.Now().Sub(created) > ctm.ttl {
		delete(ctm.tokens, token)
		return false
	}

	// Token is valid - delete it (one-time use)
	delete(ctm.tokens, token)
	return true
}

func (ctm *CSRFTokenManager) cleanup() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		ctm.mu.Lock()
		now := time.Now()
		for token, created := range ctm.tokens {
			if now.Sub(created) > ctm.ttl {
				delete(ctm.tokens, token)
			}
		}
		ctm.mu.Unlock()
	}
}

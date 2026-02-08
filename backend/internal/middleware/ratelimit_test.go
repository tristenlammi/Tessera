package middleware

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	t.Run("allows requests within limit", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Rate:     10,
			Interval: time.Minute,
			MaxBurst: 5,
		})

		// Should allow first 5 requests (burst)
		for i := 0; i < 5; i++ {
			if !rl.Allow("test-key") {
				t.Errorf("Request %d should be allowed", i+1)
			}
		}
	})

	t.Run("blocks requests exceeding limit", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Rate:     10,
			Interval: time.Minute,
			MaxBurst: 3,
		})

		// Use up the burst
		for i := 0; i < 3; i++ {
			rl.Allow("test-key")
		}

		// Next request should be blocked
		if rl.Allow("test-key") {
			t.Error("Request should be blocked after burst exhausted")
		}
	})

	t.Run("different keys have separate limits", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Rate:     10,
			Interval: time.Minute,
			MaxBurst: 2,
		})

		// Exhaust key1
		rl.Allow("key1")
		rl.Allow("key1")

		// key2 should still be allowed
		if !rl.Allow("key2") {
			t.Error("Different key should have its own limit")
		}
	})

	t.Run("tokens refill over time", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Rate:     100, // 100 per second for testing
			Interval: time.Second,
			MaxBurst: 1,
		})

		// Use the burst
		if !rl.Allow("test-key") {
			t.Error("First request should be allowed")
		}

		// Immediately blocked
		if rl.Allow("test-key") {
			t.Error("Second immediate request should be blocked")
		}

		// Wait for refill
		time.Sleep(20 * time.Millisecond) // Should refill ~2 tokens

		// Should be allowed again
		if !rl.Allow("test-key") {
			t.Error("Request after refill should be allowed")
		}
	})
}

func TestRateLimitConfig(t *testing.T) {
	t.Run("DefaultRateLimitConfig has valid values", func(t *testing.T) {
		config := DefaultRateLimitConfig()

		if config.Rate <= 0 {
			t.Error("Default rate should be positive")
		}
		if config.Interval <= 0 {
			t.Error("Default interval should be positive")
		}
		if config.MaxBurst <= 0 {
			t.Error("Default burst should be positive")
		}
		if config.KeyGenerator == nil {
			t.Error("Default key generator should not be nil")
		}
		if config.OnLimitReached == nil {
			t.Error("Default limit handler should not be nil")
		}
	})

	t.Run("StrictRateLimitConfig has stricter values", func(t *testing.T) {
		strict := StrictRateLimitConfig()
		defaultCfg := DefaultRateLimitConfig()

		if strict.Rate >= defaultCfg.Rate {
			t.Error("Strict rate should be lower than default")
		}
		if strict.MaxBurst >= defaultCfg.MaxBurst {
			t.Error("Strict burst should be lower than default")
		}
	})

	t.Run("LoginRateLimitConfig has strict values", func(t *testing.T) {
		login := LoginRateLimitConfig()

		if login.Rate > 10 {
			t.Error("Login rate should be low")
		}
		if login.MaxBurst > 5 {
			t.Error("Login burst should be low")
		}
	})
}

func TestNewRateLimiter_Defaults(t *testing.T) {
	t.Run("sets default rate if zero", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Rate:     0, // Should default
			Interval: time.Minute,
		})

		// Should work with defaults
		if !rl.Allow("test") {
			t.Error("Rate limiter with defaults should allow requests")
		}
	})

	t.Run("sets default interval if zero", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Rate:     10,
			Interval: 0, // Should default
		})

		if !rl.Allow("test") {
			t.Error("Rate limiter with defaults should allow requests")
		}
	})

	t.Run("sets default burst if zero", func(t *testing.T) {
		rl := NewRateLimiter(RateLimitConfig{
			Rate:     10,
			Interval: time.Minute,
			MaxBurst: 0, // Should default
		})

		if !rl.Allow("test") {
			t.Error("Rate limiter with defaults should allow requests")
		}
	})
}

// Benchmark tests
func BenchmarkRateLimiter_Allow(b *testing.B) {
	rl := NewRateLimiter(RateLimitConfig{
		Rate:     1000000,
		Interval: time.Second,
		MaxBurst: 1000000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow("benchmark-key")
	}
}

func BenchmarkRateLimiter_AllowDifferentKeys(b *testing.B) {
	rl := NewRateLimiter(RateLimitConfig{
		Rate:     1000000,
		Interval: time.Second,
		MaxBurst: 1000000,
	})

	keys := make([]string, 1000)
	for i := range keys {
		keys[i] = string(rune('a' + i%26))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rl.Allow(keys[i%len(keys)])
	}
}

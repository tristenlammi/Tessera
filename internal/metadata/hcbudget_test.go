package metadata

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// Hardcover allows a burst of 10 then one a second. A page that fires eight requests
// at once must not wait; a long job must settle to roughly one per second; and the
// bucket must refill while idle.
func TestBudgetIsATokenBucket(t *testing.T) {
	b := newHCBudget()
	clock := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	b.now = func() time.Time { return clock }
	b.refill = clock
	var waited []time.Duration
	b.sleep = func(d time.Duration) { waited = append(waited, d) }

	for i := 0; i < 10; i++ {
		if !b.take() {
			t.Fatal("budget refused within the daily allowance")
		}
	}
	if len(waited) != 0 {
		t.Fatalf("the first ten requests waited: %v", waited)
	}
	// Eleventh, with no time passed: one token's worth of wait (~1.1 s at 55/min).
	b.take()
	if len(waited) != 1 || waited[0] < time.Second || waited[0] > 1300*time.Millisecond {
		t.Fatalf("eleventh request waited %v, want about a second", waited)
	}
	// Twelfth: queued behind the eleventh, so about two seconds.
	b.take()
	if len(waited) != 2 || waited[1] < 2*time.Second || waited[1] > 2600*time.Millisecond {
		t.Fatalf("twelfth request waited %v, want about two seconds", waited[1])
	}
	// After a quiet half minute the burst is back.
	clock = clock.Add(30 * time.Second)
	waited = nil
	for i := 0; i < 10; i++ {
		b.take()
	}
	if len(waited) != 0 {
		t.Errorf("after idling, a fresh burst waited: %v", waited)
	}
	used, budget := b.usage()
	if used != 22 || budget != hcDailyBudget {
		t.Errorf("usage = %d/%d", used, budget)
	}
}

// A 429 is waited out and retried, honouring Retry-After; the caller sees success.
func TestQueryRetriesAfter429(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write([]byte(`{"data":{"me":[{"username":"t"}]}}`))
	}))
	defer srv.Close()
	h := NewHardcoverFunc(func() string { return "k" }, nil)
	h.endpoint = srv.URL
	var out struct {
		Me []struct {
			Username string `json:"username"`
		} `json:"me"`
	}
	start := time.Now()
	if err := h.query(context.Background(), `{ me { username } }`, nil, &out); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt32(&calls) != 2 || len(out.Me) != 1 {
		t.Errorf("calls=%d out=%+v", calls, out)
	}
	if time.Since(start) < time.Second {
		t.Error("did not wait out Retry-After before retrying")
	}
}

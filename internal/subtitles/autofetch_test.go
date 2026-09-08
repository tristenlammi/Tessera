package subtitles

import (
	"context"
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- OpenSubtitles hash ---------------------------------------------------------------

// The hash is size + the LE uint64 word sum of the first and last 64 KiB. A file of all
// zeros hashes to its size alone, which pins the arithmetic without a reference vector.
func TestOSHashOfZerosIsTheSize(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "zeros.mkv")
	size := int64(osHashChunk*3 + 12345)
	if err := os.WriteFile(p, make([]byte, size), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := osHash(p)
	if err != nil {
		t.Fatal(err)
	}
	want := "0000000000033039" // 0x33039 == 208953 == 65536*3 + 12345
	if got != want {
		t.Errorf("hash = %s, want %s", got, want)
	}
}

// Both ends of the file contribute, not just the head — the last 64 KiB is what tells a
// full download apart from a truncated one with the same opening bytes.
func TestOSHashReadsTheTail(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.mkv")
	b := filepath.Join(dir, "b.mkv")
	buf := make([]byte, osHashChunk*4)
	if err := os.WriteFile(a, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	binary.LittleEndian.PutUint64(buf[len(buf)-8:], 7)
	if err := os.WriteFile(b, buf, 0o644); err != nil {
		t.Fatal(err)
	}
	ha, _ := osHash(a)
	hb, _ := osHash(b)
	if ha == hb {
		t.Errorf("changing the last 8 bytes left the hash unchanged (%s)", ha)
	}
}

// A file smaller than two chunks can't be hashed the way the protocol expects; the caller
// searches without a hash rather than sending a wrong one.
func TestOSHashSkipsTinyFiles(t *testing.T) {
	p := filepath.Join(t.TempDir(), "tiny.mkv")
	if err := os.WriteFile(p, make([]byte, 1000), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := osHash(p)
	if err != nil || got != "" {
		t.Errorf("tiny file: hash=%q err=%v, want empty and nil", got, err)
	}
}

// --- Download sanity -------------------------------------------------------------------

// The bytes go straight to a .srt beside the media, so an HTML error page or a zip must be
// refused rather than saved.
func TestLooksLikeSRT(t *testing.T) {
	srt := []byte("1\r\n00:00:01,000 --> 00:00:03,000\r\nHello.\r\n\r\n")
	bom := append([]byte{0xEF, 0xBB, 0xBF}, srt...)
	for name, tc := range map[string]struct {
		in   []byte
		want bool
	}{
		"plain srt":      {srt, true},
		"srt with BOM":   {bom, true},
		"html error":     {[]byte("<!DOCTYPE html><html><body>Too many requests</body></html>"), false},
		"zip":            {[]byte("PK\x03\x04\x14\x00\x00\x00"), false},
		"empty":          {nil, false},
		"no timing line": {[]byte("just some text with no cues at all, long enough to pass"), false},
	} {
		if got := looksLikeSRT(tc.in); got != tc.want {
			t.Errorf("%s: looksLikeSRT = %v, want %v", name, got, tc.want)
		}
	}
}

// --- Quota -----------------------------------------------------------------------------

// Once the API says the allowance is spent, downloads pause until the reported reset —
// and unpause on their own once it passes.
func TestQuotaPausesUntilReset(t *testing.T) {
	o := NewOpenSubtitles("key", "user", "pass")
	if o.quotaPaused() {
		t.Fatal("paused before any response was seen")
	}
	reset := time.Now().Add(2 * time.Hour).UTC().Format(time.RFC3339)
	o.noteQuota(0, reset, true)
	if !o.quotaPaused() {
		t.Error("not paused after the API reported the quota spent")
	}
	if _, err := o.Download(context.Background(), "1"); err != ErrQuotaExhausted {
		t.Errorf("Download during pause = %v, want ErrQuotaExhausted without touching the network", err)
	}
	// A reset in the past means the pause is over.
	o.noteQuota(0, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339), true)
	// noteQuota falls back to +24h for a past reset time (the API said exhausted), so force
	// the clock instead to check the self-clearing path.
	o.quotaMu.Lock()
	o.quotaResetAt = time.Now().Add(-time.Second)
	o.quotaMu.Unlock()
	if o.quotaPaused() {
		t.Error("still paused after the reset time passed")
	}
	remaining, resetAt := o.Quota()
	if remaining != 0 || !resetAt.IsZero() {
		t.Errorf("Quota() = %d, %v after reset; want 0 and zero time", remaining, resetAt)
	}
}

// A successful download with downloads still remaining must not pause anything.
func TestQuotaNotPausedWhileRemaining(t *testing.T) {
	o := NewOpenSubtitles("key", "user", "pass")
	o.noteQuota(4, "", false)
	if o.quotaPaused() {
		t.Error("paused with 4 downloads remaining")
	}
	if remaining, _ := o.Quota(); remaining != 4 {
		t.Errorf("remaining = %d, want 4", remaining)
	}
}

// --- Queue -----------------------------------------------------------------------------

func queueFixture() *Service {
	return &Service{
		log:  slog.New(slog.NewTextHandler(io.Discard, nil)),
		wake: make(chan struct{}, 1),
	}
}

// The sweep re-queues every file still missing a language every six hours. Without
// dedupe, each file was in the queue as many times as sweeps had run since it last got
// to the front.
func TestEnqueueDedupesQueuedAndRunning(t *testing.T) {
	s := queueFixture()
	a := s.enqueue(&Job{Kind: "episode", SeriesID: 1, Season: 2, Episode: 3, Title: "x"})
	b := s.enqueue(&Job{Kind: "episode", SeriesID: 1, Season: 2, Episode: 3, Title: "x"})
	if a != b {
		t.Errorf("the same episode was queued twice (ids %d and %d)", a.ID, b.ID)
	}
	if s.Pending() != 1 {
		t.Errorf("pending = %d, want 1", s.Pending())
	}
	// A different episode of the same show is a different job.
	c := s.enqueue(&Job{Kind: "episode", SeriesID: 1, Season: 2, Episode: 4, Title: "y"})
	if c == a || s.Pending() != 2 {
		t.Errorf("distinct episode collapsed into the first (pending=%d)", s.Pending())
	}
	// Once a job finishes, the same file may be queued again — that's how retries work.
	s.pop()
	s.finish(a, StateSkipped, "nothing found")
	d := s.enqueue(&Job{Kind: "episode", SeriesID: 1, Season: 2, Episode: 3, Title: "x"})
	if d == a {
		t.Error("a finished job blocked re-queueing its file")
	}
}

// Queueing must never block the caller, however far behind the worker is. The old
// channel blocked after 256 — a sweep over a real library would hang the scheduler for as
// long as the worker took to drain it.
func TestEnqueueNeverBlocks(t *testing.T) {
	s := queueFixture()
	done := make(chan struct{})
	go func() {
		for i := 0; i < 5000; i++ {
			s.enqueue(&Job{Kind: "movie", MovieID: int64(i), Title: "m"})
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("enqueueing 5000 jobs with no worker running blocked")
	}
	if s.Pending() != 5000 {
		t.Errorf("pending = %d, want 5000", s.Pending())
	}
	// Jobs that haven't run are never dropped from the visible list, whatever the cap.
	queued := 0
	for _, j := range s.Jobs() {
		if j.State == StateQueued {
			queued++
		}
	}
	if queued != 5000 {
		t.Errorf("%d queued jobs visible, want all 5000 — the history cap must not drop pending work", queued)
	}
}

// The worker drains in FIFO order and sleeps when idle rather than spinning.
func TestWorkerDrainsInOrder(t *testing.T) {
	s := queueFixture()
	var got []int64
	for i := int64(1); i <= 3; i++ {
		s.enqueue(&Job{Kind: "movie", MovieID: i, Title: "m"})
	}
	for j := s.pop(); j != nil; j = s.pop() {
		got = append(got, j.MovieID)
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("drained %v, want [1 2 3]", got)
	}
	if s.pop() != nil {
		t.Error("pop on an empty queue returned a job")
	}
}

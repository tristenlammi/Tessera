package subtitles

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Removing a queued job must take it out of the worker's line without disturbing the
// order of what's left.
func TestCancelQueuedJobLeavesTheQueue(t *testing.T) {
	s := queueFixture()
	a := s.enqueue(&Job{Kind: "movie", MovieID: 1, Title: "a"})
	b := s.enqueue(&Job{Kind: "movie", MovieID: 2, Title: "b"})
	c := s.enqueue(&Job{Kind: "movie", MovieID: 3, Title: "c"})
	if err := s.Cancel(b.ID); err != nil {
		t.Fatalf("Cancel(queued) = %v", err)
	}
	if b.State != StateCancelled {
		t.Errorf("state = %q, want cancelled", b.State)
	}
	if s.Pending() != 2 {
		t.Errorf("pending = %d, want 2", s.Pending())
	}
	if got := s.pop(); got != a {
		t.Errorf("first pop = %v, want a", got)
	}
	if got := s.pop(); got != c {
		t.Errorf("second pop = %v, want c (b was removed)", got)
	}
	// A cancelled job no longer blocks the same file being queued again.
	if d := s.enqueue(&Job{Kind: "movie", MovieID: 2, Title: "b"}); d == b {
		t.Error("cancelled job stopped its file from being re-queued")
	}
}

// The Stop button's only lever on a running job is its context: cancelling it is what
// kills the whisper/ffmpeg child. The worker records the job as stopped when it returns.
func TestCancelRunningJobCancelsItsContext(t *testing.T) {
	s := queueFixture()
	job := s.enqueue(&Job{Kind: "movie", MovieID: 1, Title: "a"})
	s.pop()
	jctx, done := s.startJob(context.Background(), job)
	s.update(job, func(j *Job) { j.State = StateRunning })
	if err := s.Cancel(job.ID); err != nil {
		t.Fatalf("Cancel(running) = %v", err)
	}
	select {
	case <-jctx.Done():
	case <-time.After(time.Second):
		t.Fatal("the running job's context was not cancelled")
	}
	if job.State != StateRunning {
		t.Errorf("state flipped to %q before the worker returned; want it to stay running until process() finishes", job.State)
	}
	if job.Note != "stopping…" {
		t.Errorf("note = %q, want the UI to see it's stopping", job.Note)
	}
	done()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running != nil || s.cancelRun != nil {
		t.Error("worker bookkeeping not released after the job returned")
	}
}

func TestCancelFinishedOrUnknownJobIsRejected(t *testing.T) {
	s := queueFixture()
	job := s.enqueue(&Job{Kind: "movie", MovieID: 1, Title: "a"})
	s.pop()
	s.finish(job, StateDone, "1 extracted")
	if err := s.Cancel(job.ID); !errors.Is(err, ErrJobNotActive) {
		t.Errorf("Cancel(done) = %v, want ErrJobNotActive", err)
	}
	if err := s.Cancel(999); !errors.Is(err, ErrJobNotActive) {
		t.Errorf("Cancel(unknown) = %v, want ErrJobNotActive", err)
	}
}

// Clear drops what hasn't started; the job the worker is on is the Stop button's business.
func TestClearQueueSparesTheRunningJob(t *testing.T) {
	s := queueFixture()
	first := s.enqueue(&Job{Kind: "movie", MovieID: 1, Title: "a"})
	s.enqueue(&Job{Kind: "movie", MovieID: 2, Title: "b"})
	s.enqueue(&Job{Kind: "movie", MovieID: 3, Title: "c"})
	s.pop()
	_, done := s.startJob(context.Background(), first)
	defer done()
	s.update(first, func(j *Job) { j.State = StateRunning })
	if n := s.ClearQueue(); n != 2 {
		t.Errorf("ClearQueue = %d, want 2", n)
	}
	if s.Pending() != 0 {
		t.Errorf("pending = %d after clear", s.Pending())
	}
	if first.State != StateRunning {
		t.Errorf("running job state = %q, clear must leave it alone", first.State)
	}
	cancelled := 0
	for _, j := range s.Jobs() {
		if j.State == StateCancelled {
			cancelled++
		}
	}
	if cancelled != 2 {
		t.Errorf("%d jobs shown as cancelled, want 2", cancelled)
	}
}

// A finished job must not keep showing the stage/percent it had while running.
func TestFinishClearsInFlightProgress(t *testing.T) {
	s := queueFixture()
	job := s.enqueue(&Job{Kind: "movie", MovieID: 1, Title: "a"})
	s.update(job, func(j *Job) { j.State = StateRunning; j.Stage = "AI transcribing (en)"; j.Progress = 40 })
	s.finish(job, StateCancelled, "stopped")
	if job.Stage != "" || job.Progress != 0 {
		t.Errorf("stage=%q progress=%d after finish, want both cleared", job.Stage, job.Progress)
	}
}

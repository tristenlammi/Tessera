package subtitles

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// JobState is the lifecycle of a subtitle-ensure job.
type JobState string

const (
	StateQueued  JobState = "queued"
	StateRunning JobState = "running"
	StateDone    JobState = "done"
	StateSkipped JobState = "skipped"
	StateFailed  JobState = "failed"
)

// Job is one file's "make sure the kept-language subtitles exist" task — the unit the Queue tab
// shows. Extraction and downloads happen inside process().
type Job struct {
	ID       int64    `json:"id"`
	Kind     string   `json:"kind"` // "movie" | "episode"
	MovieID  int64    `json:"movie_id,omitempty"`
	SeriesID int64    `json:"series_id,omitempty"`
	Season   int      `json:"season,omitempty"`
	Episode  int      `json:"episode,omitempty"`
	Title    string   `json:"title"`
	State    JobState `json:"state"`
	Note     string   `json:"note,omitempty"`
	At       int64    `json:"at"` // unix seconds queued
	// Progress is 0-100 while an AI run is in flight (extraction and downloads are too
	// quick to bother). Stays 0 for everything else.
	Progress int    `json:"progress,omitempty"`
	Stage    string `json:"stage,omitempty"` // what the running job is doing right now
}

// key identifies the file a job is for, so the same file can't be queued twice.
func (j *Job) key() string {
	if j.Kind == "episode" {
		return fmt.Sprintf("e:%d:%d:%d", j.SeriesID, j.Season, j.Episode)
	}
	return fmt.Sprintf("m:%d", j.MovieID)
}

// LogLine is one entry in the Subtitles activity console.
type LogLine struct {
	At    int64  `json:"at"`
	Level string `json:"level"` // "info" | "warn" | "error"
	Msg   string `json:"msg"`
}

// Run drains the queue in a single worker until ctx is cancelled (start it in a goroutine).
//
// The queue is a slice under the mutex rather than a channel. A channel send blocks once
// the buffer is full, and the one worker can be inside a whisper run for many minutes —
// so a sweep that queued a few hundred files would block the scheduler goroutine, and an
// import hook would block the import, for as long as it took the worker to drain. A
// slice grows; the caller always returns immediately.
func (s *Service) Run(ctx context.Context) {
	s.log.Info("subtitles: worker started")
	for {
		job := s.pop()
		if job == nil {
			select {
			case <-ctx.Done():
				return
			case <-s.wake:
			}
			continue
		}
		s.process(ctx, job)
	}
}

// pop takes the oldest pending job, or nil.
func (s *Service) pop() *Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.pending) == 0 {
		return nil
	}
	job := s.pending[0]
	s.pending = s.pending[1:]
	return job
}

// enqueue registers a job and wakes the worker. Returns the existing job instead when the
// same file is already queued or running: the 6-hourly sweep re-queues everything still
// missing, and without this every file was in the queue several times over.
func (s *Service) enqueue(job *Job) *Job {
	s.mu.Lock()
	key := job.key()
	for _, j := range s.jobs {
		if (j.State == StateQueued || j.State == StateRunning) && j.key() == key {
			s.mu.Unlock()
			return j
		}
	}
	s.nextID++
	job.ID = s.nextID
	job.State = StateQueued
	job.At = time.Now().Unix()
	s.jobs = append([]*Job{job}, s.jobs...)
	// Keep the visible history bounded, but never drop a job that hasn't run yet.
	if len(s.jobs) > 200 {
		kept := s.jobs[:0:0]
		for i, j := range s.jobs {
			if i < 200 || j.State == StateQueued || j.State == StateRunning {
				kept = append(kept, j)
			}
		}
		s.jobs = kept
	}
	s.pending = append(s.pending, job)
	s.mu.Unlock()
	s.event("info", "Queued "+job.Title)
	select {
	case s.wake <- struct{}{}:
	default: // worker already awake
	}
	return job
}

// QueueMovie enqueues a subtitle-ensure job for one movie.
func (s *Service) QueueMovie(ctx context.Context, movieID int64) (*Job, error) {
	m, err := s.movies.Get(ctx, movieID)
	if err != nil {
		return nil, err
	}
	if !m.HasFile || m.MovieFilePath == "" {
		return nil, fmt.Errorf("movie has no file")
	}
	return s.enqueue(&Job{Kind: "movie", MovieID: movieID, Title: m.Title}), nil
}

// QueueEpisode enqueues a subtitle-ensure job for one TV episode.
func (s *Service) QueueEpisode(ctx context.Context, seriesID int64, season, episode int) (*Job, error) {
	path, _ := s.series.EpisodeFilePath(ctx, seriesID, season, episode)
	if path == "" {
		return nil, fmt.Errorf("episode has no file")
	}
	title := fmt.Sprintf("S%02dE%02d", season, episode)
	if sm, err := s.series.Get(ctx, seriesID); err == nil {
		title = fmt.Sprintf("%s - S%02dE%02d", sm.Title, season, episode)
	}
	return s.enqueue(&Job{Kind: "episode", SeriesID: seriesID, Season: season, Episode: episode, Title: title}), nil
}

// OnMovieImported is the import hook: a movie that just landed gets its subtitles ensured
// now, rather than whenever the next 6-hourly sweep happens to run.
func (s *Service) OnMovieImported(ctx context.Context, movieID int64) {
	if !s.settings.GetBool(ctx, keyMoviesAuto, defaultMoviesAuto) {
		return
	}
	if _, err := s.QueueMovie(ctx, movieID); err != nil {
		s.log.Debug("subtitles: import hook skipped movie", "movie_id", movieID, "err", err)
	}
}

// OnSeriesImported is the import hook for TV. The import only reports the series, not
// which episodes it wrote, so every episode of that show still missing a kept language is
// queued — for a fresh import that is the new episode(s), and anything older that was
// missing gets picked up along the way. Sidecar check only; nothing is probed.
func (s *Service) OnSeriesImported(ctx context.Context, seriesID int64) {
	if !s.settings.GetBool(ctx, keySeriesAuto, defaultSeriesAuto) {
		return
	}
	n := 0
	for _, e := range s.missingEpisodes(ctx, seriesID) {
		if _, err := s.QueueEpisode(ctx, seriesID, e.season, e.episode); err == nil {
			n++
		}
	}
	if n > 0 {
		s.log.Info("subtitles: import hook queued episodes", "series_id", seriesID, "count", n)
	}
}

// SweepMissing enqueues an ensure job for every downloaded file still missing a kept-language
// subtitle (media = "movies" | "tv"). Returns how many jobs were queued.
//
// Decides "missing" from the sidecars on disk alone. It used to go through Library(),
// which probes every video file for embedded tracks — a full ffprobe pass over the whole
// catalogue, every six hours, to answer a question a directory listing answers.
func (s *Service) SweepMissing(ctx context.Context, media string) (int, error) {
	n := 0
	if media == "tv" {
		list, err := s.series.List(ctx)
		if err != nil {
			return 0, err
		}
		for _, sm := range list {
			if ctx.Err() != nil {
				return n, ctx.Err()
			}
			for _, e := range s.missingEpisodes(ctx, sm.ID) {
				if _, err := s.QueueEpisode(ctx, sm.ID, e.season, e.episode); err == nil {
					n++
				}
			}
		}
		return n, nil
	}
	list, err := s.movies.List(ctx)
	if err != nil {
		return 0, err
	}
	langs := s.languages(ctx)
	for _, m := range list {
		if ctx.Err() != nil {
			return n, ctx.Err()
		}
		if !m.HasFile || m.MovieFilePath == "" {
			continue
		}
		if len(missingOf(langs, presentLanguages(m.MovieFilePath, langs, true))) == 0 {
			continue
		}
		if _, err := s.QueueMovie(ctx, m.ID); err == nil {
			n++
		}
	}
	return n, nil
}

type epRef struct{ season, episode int }

// missingEpisodes lists one show's episodes that have a file but lack a kept language.
func (s *Service) missingEpisodes(ctx context.Context, seriesID int64) []epRef {
	full, err := s.series.Get(ctx, seriesID)
	if err != nil {
		return nil
	}
	langs := s.languages(ctx)
	var out []epRef
	for _, sn := range full.Seasons {
		for _, e := range sn.Episodes {
			if !e.HasFile || e.FilePath == "" {
				continue
			}
			if len(missingOf(langs, presentLanguages(e.FilePath, langs, false))) == 0 {
				continue
			}
			out = append(out, epRef{e.SeasonNumber, e.EpisodeNumber})
		}
	}
	return out
}

// Jobs returns a snapshot of recent jobs (newest first).
func (s *Service) Jobs() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Job, len(s.jobs))
	for i, j := range s.jobs {
		out[i] = *j
	}
	return out
}

// Pending reports how many jobs are waiting, for the status line.
func (s *Service) Pending() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.pending)
}

// update mutates a job under lock.
func (s *Service) update(job *Job, fn func(*Job)) {
	s.mu.Lock()
	fn(job)
	s.mu.Unlock()
}

// finish sets a job's terminal state + note.
func (s *Service) finish(job *Job, state JobState, note string) {
	s.update(job, func(j *Job) { j.State = state; j.Note = note })
}

// event appends a line to the activity console (kept to the last 500) and mirrors it to the log.
func (s *Service) event(level, msg string) {
	s.logMu.Lock()
	s.logBuf = append(s.logBuf, LogLine{At: time.Now().Unix(), Level: level, Msg: msg})
	if len(s.logBuf) > 500 {
		s.logBuf = s.logBuf[len(s.logBuf)-500:]
	}
	s.logMu.Unlock()
	if level == "warn" || level == "error" {
		s.log.Warn("subtitles: " + msg)
	} else {
		s.log.Info("subtitles: " + msg)
	}
}

// Logs returns the recent activity console lines (oldest first).
func (s *Service) Logs() []LogLine {
	s.logMu.Lock()
	defer s.logMu.Unlock()
	out := make([]LogLine, len(s.logBuf))
	copy(out, s.logBuf)
	return out
}

// missingOf returns the wanted languages that aren't present (case-insensitive).
func missingOf(wanted, present []string) []string {
	have := make(map[string]bool, len(present))
	for _, p := range present {
		have[strings.ToLower(p)] = true
	}
	var out []string
	for _, w := range wanted {
		if !have[strings.ToLower(w)] {
			out = append(out, w)
		}
	}
	return out
}

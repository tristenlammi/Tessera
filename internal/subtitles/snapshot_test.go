package subtitles

import (
	"testing"
	"time"
)

// The Overview reads totals from the last pass. Before the first pass completes there are
// no figures, and the response has to say so rather than report an empty library.
func TestCoverageBeforeAndAfterAPass(t *testing.T) {
	s := queueFixture()
	c := s.Coverage()
	if c.ScannedAt != 0 || c.Files != 0 {
		t.Errorf("before any pass: %+v, want no figures", c)
	}
	if _, scanned := s.SnapshotMovies(); scanned {
		t.Error("SnapshotMovies reported a completed pass before one ran")
	}

	s.snap.mu.Lock()
	s.snap.movies = []FileSubs{
		{Kind: "movie", MovieID: 1, Missing: 0},
		{Kind: "movie", MovieID: 2, Missing: 1},
		{Kind: "movie", MovieID: 3, Missing: 0},
	}
	s.snap.groups = []SeriesGroup{
		{SeriesID: 10, Episodes: 10, Covered: 4, Missing: 6},
		{SeriesID: 11, Episodes: 3, Covered: 3, Missing: 0},
	}
	s.snap.at = time.Now()
	s.snap.mu.Unlock()

	c = s.Coverage()
	if c.ScannedAt == 0 {
		t.Error("ScannedAt not set after a pass")
	}
	if c.Movies != (CoverageCounts{Files: 3, Covered: 2, Missing: 1}) {
		t.Errorf("movies = %+v", c.Movies)
	}
	if c.TV != (CoverageCounts{Files: 13, Covered: 7, Missing: 6}) {
		t.Errorf("tv = %+v", c.TV)
	}
	if c.Files != 16 || c.Covered != 9 || c.Missing != 7 {
		t.Errorf("totals = %d/%d/%d, want 16/9/7", c.Files, c.Covered, c.Missing)
	}
	if got, scanned := s.SnapshotGroups(); !scanned || len(got) != 2 {
		t.Errorf("SnapshotGroups = %d groups, scanned=%v", len(got), scanned)
	}
}

// A fixture with no catalogs can't scan; Rescan must say so instead of dereferencing nil
// in a goroutine. (The worker calls it unconditionally at startup.)
func TestRescanWithoutCatalogsIsANoop(t *testing.T) {
	s := queueFixture()
	if s.Rescan(t.Context()) {
		t.Error("Rescan started a pass with no movie/series catalogs to read")
	}
	if s.Coverage().Scanning {
		t.Error("left the snapshot marked as scanning")
	}
}

func TestFmtAudio(t *testing.T) {
	for sec, want := range map[float64]string{
		42 * 60:       "42m",
		63 * 60:       "1h 03m",
		2*3600 + 5*60: "2h 05m",
		30:            "1m", // rounds to the nearest minute; "0m" would read as nothing
	} {
		if got := fmtAudio(sec); got != want {
			t.Errorf("fmtAudio(%v) = %q, want %q", sec, got, want)
		}
	}
}

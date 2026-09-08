package subtitles

import (
	"testing"
	"time"
)

const silenceOut = `[silencedetect @ 0x1] silence_start: 0
[silencedetect @ 0x1] silence_end: 12.5 | silence_duration: 12.5
[silencedetect @ 0x1] silence_start: 130.2
[silencedetect @ 0x1] silence_end: 132.0 | silence_duration: 1.8
[silencedetect @ 0x1] silence_start: 250.7
[silencedetect @ 0x1] silence_end: 253.1 | silence_duration: 2.4
[silencedetect @ 0x1] silence_start: 600.0
[silencedetect @ 0x1] silence_end: 601.5 | silence_duration: 1.5
[silencedetect @ 0x1] silence_start: 1790.0
`

func TestParseSilencesPairsAndClosesTrailing(t *testing.T) {
	sils := parseSilences(silenceOut, 1800*time.Second)
	if len(sils) != 5 {
		t.Fatalf("got %d silences, want 5: %+v", len(sils), sils)
	}
	if sils[0].start != 0 || sils[0].end != 12500*time.Millisecond {
		t.Errorf("first silence = %+v", sils[0])
	}
	if sils[4].start != 1790*time.Second || sils[4].end != 1800*time.Second {
		t.Errorf("trailing silence not closed at the file end: %+v", sils[4])
	}
}

// Chunks skip the leading silence, cut at the first qualifying pause past the target,
// leave most of each pause out, and never split anywhere but inside a pause.
func TestPlanChunksCutsInsidePauses(t *testing.T) {
	total := 1800 * time.Second
	sils := parseSilences(silenceOut, total)
	chunks := planChunks(total, sils)
	if len(chunks) < 3 {
		t.Fatalf("got %d chunks: %+v", len(chunks), chunks)
	}
	// Leading 12.5 s of silence is skipped (minus the pad).
	if want := 12500*time.Millisecond - chunkSilencePad; chunks[0].start != want {
		t.Errorf("first chunk starts at %v, want %v (after the leading silence)", chunks[0].start, want)
	}
	// 130.2 s and 250.7 s are both short of the 4-minute target; the next pause (600 s)
	// is past the 8-minute maximum, so the cut falls back to the 250.7 s pause.
	if want := 250700*time.Millisecond + chunkSilencePad; chunks[0].end != want {
		t.Errorf("first chunk ends at %v, want %v", chunks[0].end, want)
	}
	if want := 253100*time.Millisecond - chunkSilencePad; chunks[1].start != want {
		t.Errorf("second chunk starts at %v, want %v", chunks[1].start, want)
	}
	// Every boundary except the ends lies inside a detected silence.
	inside := func(at time.Duration) bool {
		for _, s := range sils {
			if at >= s.start && at <= s.end {
				return true
			}
		}
		return false
	}
	for i, c := range chunks {
		if c.end <= c.start {
			t.Errorf("chunk %d is empty: %+v", i, c)
		}
		if i > 0 && !inside(c.start) {
			t.Errorf("chunk %d starts outside a pause: %v", i, c.start)
		}
		if i < len(chunks)-1 && !inside(c.end) {
			t.Errorf("chunk %d ends outside a pause: %v", i, c.end)
		}
	}
	// The trailing silence (1790 s to the end) is left out, like any other pause.
	if last := chunks[len(chunks)-1]; last.end != 1790*time.Second+chunkSilencePad {
		t.Errorf("last chunk ends at %v, want just inside the trailing silence", last.end)
	}
}

// With no pauses at all the file is one chunk; a short file is one chunk.
func TestPlanChunksWithoutPauses(t *testing.T) {
	if c := planChunks(20*time.Minute, nil); len(c) != 1 || c[0].start != 0 || c[0].end != 20*time.Minute {
		t.Errorf("no pauses: %+v", c)
	}
	if c := planChunks(90*time.Second, []silence{{40 * time.Second, 42 * time.Second}}); len(c) != 1 {
		t.Errorf("short file was split: %+v", c)
	}
}

// A long stretch with only sub-target pauses gets cut at the last decent pause before
// the maximum, not left to run on.
func TestPlanChunksUsesFallbackPauseBeforeMax(t *testing.T) {
	var sils []silence
	// A pause every 2.5 minutes — none reaches the 4-minute target on its own.
	for at := 150 * time.Second; at < 30*time.Minute; at += 150 * time.Second {
		sils = append(sils, silence{at, at + 2*time.Second})
	}
	chunks := planChunks(30*time.Minute, sils)
	for i, c := range chunks {
		if d := c.end - c.start; d > chunkMax+chunkSilencePad {
			t.Errorf("chunk %d is %v long, over the maximum", i, d)
		}
	}
	if len(chunks) < 4 {
		t.Errorf("only %d chunks for 30 minutes: %+v", len(chunks), chunks)
	}
}

func TestOverallProgress(t *testing.T) {
	c := chunk{10 * time.Minute, 14 * time.Minute}
	if p := overallProgress(c, 50, 40*time.Minute); p != 30 {
		t.Errorf("halfway through the 10–14 min chunk of a 40 min file = %d%%, want 30", p)
	}
	if p := overallProgress(c, 100, 40*time.Minute); p != 35 {
		t.Errorf("end of chunk = %d%%, want 35", p)
	}
	if p := overallProgress(c, 50, 0); p != 50 {
		t.Errorf("unknown total should pass the chunk's own percent through, got %d", p)
	}
}

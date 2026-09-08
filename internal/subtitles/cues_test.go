package subtitles

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

const longSeg = `1
00:01:00,000 --> 00:01:08,000
I told him we'd be there by nine, but the traffic on the bridge was worse than anyone expected, so we missed the whole first act.

`

// A whisper segment that runs past two lines is split at clause boundaries, the pieces
// stay in order with no words lost, and their timing tiles the segment's exactly.
func TestSplitLongSegmentAtPunctuation(t *testing.T) {
	cues := parseSRT(longSeg)
	if len(cues) != 1 {
		t.Fatalf("parsed %d cues, want 1", len(cues))
	}
	pieces := splitCue(cues[0])
	if len(pieces) < 2 {
		t.Fatalf("a %d-char segment was not split", utf8.RuneCountInString(cues[0].text))
	}
	var joined []string
	for i, p := range pieces {
		if n := utf8.RuneCountInString(p.text); n > cueMaxChars {
			t.Errorf("piece %d is %d chars: %q", i, n, p.text)
		}
		if i > 0 && p.start != pieces[i-1].end {
			t.Errorf("piece %d starts at %v, previous ended at %v", i, p.start, pieces[i-1].end)
		}
		joined = append(joined, p.text)
	}
	if got := strings.Join(joined, " "); got != cues[0].text {
		t.Errorf("words changed:\n got %q\nwant %q", got, cues[0].text)
	}
	if pieces[0].start != cues[0].start || pieces[len(pieces)-1].end != cues[0].end {
		t.Error("pieces don't span the original segment")
	}
	// The first cut lands after a comma, not mid-clause.
	if !endsClause(strings.Fields(pieces[0].text)[len(strings.Fields(pieces[0].text))-1]) {
		t.Errorf("first piece doesn't end at a clause boundary: %q", pieces[0].text)
	}
	// Time is shared by length: the longer piece gets the longer slice.
	for i := 1; i < len(pieces); i++ {
		a, b := pieces[i-1], pieces[i]
		la, lb := utf8.RuneCountInString(a.text), utf8.RuneCountInString(b.text)
		if (la > lb*2 && a.end-a.start < b.end-b.start) || (lb > la*2 && b.end-b.start < a.end-a.start) {
			t.Errorf("time not proportional: %q %v vs %q %v", a.text, a.end-a.start, b.text, b.end-b.start)
		}
	}
}

// A one- or two-word cue that follows straight on from the previous one is the same
// utterance and joins it; across a real pause it stands alone.
func TestFragmentsMergeOnlyAcrossSmallGaps(t *testing.T) {
	cues := []cue{
		{0, 2 * time.Second, "We should go now."},
		{2100 * time.Millisecond, 2500 * time.Millisecond, "Right?"},
		{6 * time.Second, 7 * time.Second, "Okay."},
		{7100 * time.Millisecond, 9 * time.Second, "Let me get my coat and we'll leave."},
	}
	got := mergeFragments(cues)
	if len(got) != 2 {
		t.Fatalf("got %d cues, want 2: %+v", len(got), got)
	}
	if got[0].text != "We should go now. Right?" || got[0].end != 2500*time.Millisecond {
		t.Errorf("first merge wrong: %+v", got[0])
	}
	if got[1].text != "Okay. Let me get my coat and we'll leave." || got[1].start != 6*time.Second {
		t.Errorf("forward merge wrong: %+v", got[1])
	}
	// A long pause keeps a fragment separate.
	apart := mergeFragments([]cue{{0, time.Second, "Hey."}, {5 * time.Second, 7 * time.Second, "Did you hear that noise outside?"}})
	if len(apart) != 2 {
		t.Errorf("merged across a 4s pause: %+v", apart)
	}
}

// Nothing flashes: a cue stays up at least a second, but never into the next one.
func TestFloorDurations(t *testing.T) {
	cues := []cue{
		{0, 300 * time.Millisecond, "Hi."},
		{600 * time.Millisecond, 3 * time.Second, "Long enough already."},
		{10 * time.Second, 10200 * time.Millisecond, "Bye."},
	}
	floorDurations(cues)
	if want := 600*time.Millisecond - cueGuardGap; cues[0].end != want {
		t.Errorf("first cue end = %v, want %v (stopped short of the next cue)", cues[0].end, want)
	}
	if cues[1].end != 3*time.Second {
		t.Errorf("a long-enough cue was changed: %v", cues[1].end)
	}
	if cues[2].end != 11*time.Second {
		t.Errorf("last cue end = %v, want a full second", cues[2].end)
	}
}

func TestLayoutLinesBreaksNearTheMiddle(t *testing.T) {
	if got := layoutLines("Short enough for one line."); strings.Contains(got, "\n") {
		t.Errorf("one-line text was broken: %q", got)
	}
	got := layoutLines("I told him we'd be there by nine, but the traffic was worse than expected.")
	lines := strings.Split(got, "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %q", got)
	}
	for _, l := range lines {
		if utf8.RuneCountInString(l) > cueLineChars {
			t.Errorf("line over %d chars: %q", cueLineChars, l)
		}
	}
	if !strings.HasSuffix(lines[0], "nine,") {
		t.Errorf("didn't break after the comma: %q", lines[0])
	}
}

func TestSRTTimesRoundTrip(t *testing.T) {
	in := "1\n01:02:03,450 --> 01:02:05,000\nHello there.\n\n"
	cues := parseSRT(in)
	if len(cues) != 1 || cues[0].start != time.Hour+2*time.Minute+3*time.Second+450*time.Millisecond {
		t.Fatalf("parsed %+v", cues)
	}
	if got := formatSRT(cues); got != in {
		t.Errorf("round trip changed the file:\n%q\n%q", got, in)
	}
	if fmtSRTTime(-time.Second) != "00:00:00,000" {
		t.Error("negative time should clamp to zero")
	}
}

// splitCue plus the shared passes give valid, numbered SRT with every line within the
// limits, and leave an ordinary sentence as it was.
func TestSplitAndLayoutEndToEnd(t *testing.T) {
	in := "1\n00:00:01,000 --> 00:00:03,000\nA perfectly ordinary sentence.\n\n2\n" + longSeg[2:]
	var shaped []cue
	for _, c := range parseSRT(in) {
		shaped = append(shaped, splitCue(c)...)
	}
	shaped = mergeFragments(shaped)
	floorDurations(shaped)
	for i := range shaped {
		shaped[i].text = layoutLines(shaped[i].text)
	}
	out := formatSRT(shaped)
	cues := parseSRT(out)
	if len(cues) < 3 {
		t.Fatalf("got %d cues:\n%s", len(cues), out)
	}
	if cues[0].text != "A perfectly ordinary sentence." || cues[0].start != time.Second || cues[0].end != 3*time.Second {
		t.Errorf("ordinary cue was altered: %+v", cues[0])
	}
	for _, l := range strings.Split(out, "\n") {
		if srtIndexLine.MatchString(l) || srtTimeRe.MatchString(l) {
			continue
		}
		if utf8.RuneCountInString(l) > cueLineChars {
			t.Errorf("line over limit: %q", l)
		}
	}
	for i, c := range cues {
		if c.end-c.start < cueMinDur-cueGuardGap {
			t.Errorf("cue %d on screen only %v", i+1, c.end-c.start)
		}
	}
	if len(parseSRT("garbage")) != 0 {
		t.Error("non-SRT input parsed as cues")
	}
}

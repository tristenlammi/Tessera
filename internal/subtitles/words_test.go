package subtitles

import (
	"math"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A segment as whisper-cli -ojf -dtw writes it: t_dtw in centiseconds, offsets in ms,
// tokens as BPE pieces with a leading space on word starts, control tokens mixed in.
const dtwSegment = `{"transcription":[{"timestamps":{"from":"00:00:00,000","to":"00:00:06,000"},"offsets":{"from":0,"to":6000},"text":" We should go now. Right?","tokens":[
{"text":"[_BEG_]","offsets":{"from":0,"to":0},"id":50364,"p":0.9,"t_dtw":-1},
{"text":" We","offsets":{"from":0,"to":300},"id":1,"p":0.9,"t_dtw":320},
{"text":" should","offsets":{"from":300,"to":600},"id":2,"p":0.9,"t_dtw":355},
{"text":" go","offsets":{"from":600,"to":900},"id":3,"p":0.9,"t_dtw":372},
{"text":" now","offsets":{"from":900,"to":1200},"id":4,"p":0.9,"t_dtw":398},
{"text":".","offsets":{"from":1200,"to":1300},"id":5,"p":0.9,"t_dtw":402},
{"text":" Right","offsets":{"from":1300,"to":1600},"id":6,"p":0.9,"t_dtw":540},
{"text":"?","offsets":{"from":1600,"to":1700},"id":7,"p":0.9,"t_dtw":548},
{"text":"[_TT_300]","offsets":{"from":0,"to":0},"id":50664,"p":0.9,"t_dtw":-1}
]}]}`

// The segment says 0–6 s but the speech is at 3.2–5.5 s; the words must follow the
// DTW times, not the segment, with the chunk's start added.
func TestWordsFromJSONUsesDTWTimes(t *testing.T) {
	base := 10 * time.Minute
	words, err := wordsFromJSON([]byte(dtwSegment), base)
	if err != nil {
		t.Fatal(err)
	}
	texts := make([]string, 0, len(words))
	for _, w := range words {
		texts = append(texts, w.text)
	}
	if got := strings.Join(texts, " "); got != "We should go now. Right?" {
		t.Errorf("words = %q", got)
	}
	// Punctuation attached to its word; control tokens gone.
	if words[3].text != "now." || words[4].text != "Right?" {
		t.Errorf("punctuation not attached: %q %q", words[3].text, words[4].text)
	}
	if want := base + 3200*time.Millisecond; words[0].end != want {
		t.Errorf("first word ends at %v, want %v (t_dtw 320 cs + chunk start)", words[0].end, want)
	}
	// The first word's start is bounded by its own length, not the segment's 0.
	if want := base + 3200*time.Millisecond - wordMaxDur; words[0].start != want {
		t.Errorf("first word starts at %v, want %v — segment start must not pull it early", words[0].start, want)
	}
	// Each following word starts where the previous ended.
	for i := 1; i < len(words); i++ {
		if words[i].start != words[i-1].end {
			t.Errorf("word %d starts at %v, previous ended at %v", i, words[i].start, words[i-1].end)
		}
	}
	if want := base + 5480*time.Millisecond; words[4].end != want {
		t.Errorf("last word ends at %v, want %v", words[4].end, want)
	}
}

// Without DTW (t_dtw -1, or values that don't land in the segment) the words share the
// segment's span by length — the pre-DTW behaviour, never a silent 10× shift.
func TestWordsFromJSONFallsBackWithoutDTW(t *testing.T) {
	noDTW := strings.ReplaceAll(dtwSegment, `"t_dtw":3`, `"t_dtw":-1`)
	noDTW = strings.ReplaceAll(noDTW, `"t_dtw":4`, `"t_dtw":-1`)
	noDTW = strings.ReplaceAll(noDTW, `"t_dtw":5`, `"t_dtw":-1`)
	words, err := wordsFromJSON([]byte(noDTW), 0)
	if err != nil {
		t.Fatal(err)
	}
	if words[0].start != 0 || words[len(words)-1].end != 6*time.Second {
		t.Errorf("fallback words don't span the segment: %+v", words)
	}
	// Values that would put the words far outside the segment are distrusted too.
	wild := strings.ReplaceAll(dtwSegment, `"t_dtw":540`, `"t_dtw":540000`)
	words, _ = wordsFromJSON([]byte(wild), 0)
	if words[len(words)-1].end != 6*time.Second {
		t.Errorf("implausible t_dtw was trusted: last word ends at %v", words[len(words)-1].end)
	}
}

func TestDTWUnitDetectsMilliseconds(t *testing.T) {
	seg := wsSegment{Offsets: wsOffsets{From: 0, To: 6000}, Tokens: []wsToken{
		{Text: " We", TDTW: 3200}, {Text: " go", TDTW: 5400},
	}}
	if u := dtwUnit(seg, 0, 6*time.Second); u != time.Millisecond {
		t.Errorf("unit = %v, want ms for values that only fit as ms", u)
	}
}

// wordsAt builds words from "text@start-end" specs (seconds).
func wordsAt(spec ...string) []word {
	var out []word
	for _, s := range spec {
		at := strings.LastIndex(s, "@")
		se := strings.SplitN(s[at+1:], "-", 2)
		a, err := strconv.ParseFloat(se[0], 64)
		if err != nil {
			panic(err)
		}
		b, err := strconv.ParseFloat(se[1], 64)
		if err != nil {
			panic(err)
		}
		out = append(out, word{time.Duration(math.Round(a*1000)) * time.Millisecond, time.Duration(math.Round(b*1000)) * time.Millisecond, s[:at]})
	}
	return out
}

// Cues break at pauses and at sentence ends, and a cue is timed by its words — the
// second sentence here starts at 5.0 s, not at the segment's 0.
func TestCuesFromWordsBreakAtPausesAndSentences(t *testing.T) {
	words := wordsAt(
		"We@0.5-0.7", "should@0.7-0.9", "go@0.9-1.1", "now.@1.1-1.4",
		"Right?@1.6-1.9",
		// 3 s pause
		"He'll@5.0-5.2", "come@5.2-5.4", "when@5.4-5.5", "he's@5.5-5.7", "ready,@5.7-6.0",
		"and@6.1-6.2", "if@6.2-6.3", "he@6.3-6.4", "doesn't@6.4-6.7", "we@6.7-6.8", "know@6.8-7.0", "where@7.0-7.2", "we@7.2-7.3", "stand.@7.3-7.7",
		"Okay.@8.2-8.5",
	)
	cues := cuesFromWords(words)
	if len(cues) < 3 {
		t.Fatalf("got %d cues: %+v", len(cues), cues)
	}
	if cues[0].text != "We should go now. Right?" {
		t.Errorf("first cue = %q — a short sentence should not break on its own", cues[0].text)
	}
	if cues[1].start != 5*time.Second {
		t.Errorf("second cue starts at %v, want 5s (its first word)", cues[1].start)
	}
	if !strings.HasPrefix(cues[1].text, "He'll come") {
		t.Errorf("second cue = %q", cues[1].text)
	}
	// "stand." ends a long sentence and "Okay." follows after a 500 ms pause → break.
	last := cues[len(cues)-1]
	if last.text != "Okay." || last.start != 8200*time.Millisecond {
		t.Errorf("last cue = %+v, want Okay. at 8.2s", last)
	}
}

// Reading time: a two-line cue is held long enough to read, but never into the next.
func TestPadCuesHoldsForReading(t *testing.T) {
	cues := []cue{
		{2 * time.Second, 2500 * time.Millisecond, "I told him we'd be there by nine but the traffic was worse than anyone expected."},
		{9 * time.Second, 9200 * time.Millisecond, "Yes."},
		{9500 * time.Millisecond, 10 * time.Second, "No."},
	}
	padCues(cues)
	if cues[0].end < 5*time.Second {
		t.Errorf("an 80-char cue released after %v — too fast to read", cues[0].end-cues[0].start)
	}
	if want := 9500*time.Millisecond - cueGuardGap; cues[1].end != want {
		t.Errorf("second cue ends at %v, want %v (stopped short of the next)", cues[1].end, want)
	}
	if cues[2].end != 10500*time.Millisecond {
		t.Errorf("last cue held %v, want a full second", cues[2].end-cues[2].start)
	}
}

func TestDropRepeatsRemovesLoops(t *testing.T) {
	mk := func(texts ...string) []cue {
		var out []cue
		for i, s := range texts {
			out = append(out, cue{time.Duration(i) * time.Second, time.Duration(i+1) * time.Second, s})
		}
		return out
	}
	got := dropRepeats(mk("Hello.", "I'm not saying this is how I'm going to be.", "I'm not saying this is how I'm going to be.",
		"I'm not saying this is how I'm going to be.", "I'm not saying this is how I'm going to be.", "Goodbye."))
	if len(got) != 3 || got[1].text != "I'm not saying this is how I'm going to be." || got[2].text != "Goodbye." {
		t.Errorf("loop not collapsed to one line: %+v", got)
	}
	// A line said twice is legitimate ("No. No.") and stays.
	if got := dropRepeats(mk("No.", "No.", "Stop it.")); len(got) != 3 {
		t.Errorf("a doubled line was removed: %+v", got)
	}
}

func TestShapeWordCuesEndToEnd(t *testing.T) {
	words, err := wordsFromJSON([]byte(dtwSegment), 0)
	if err != nil {
		t.Fatal(err)
	}
	cues := shapeWordCues(words)
	if len(cues) != 1 {
		t.Fatalf("got %d cues: %+v", len(cues), cues)
	}
	c := cues[0]
	if c.text != "We should go now. Right?" {
		t.Errorf("text = %q", c.text)
	}
	if c.start != 3200*time.Millisecond-wordMaxDur || c.end < 5480*time.Millisecond+cueEndPad {
		t.Errorf("cue timed %v–%v, want ~2.4s to ≥5.73s", c.start, c.end)
	}
	if !strings.Contains(formatSRT(cues), "00:00:02,400 --> ") {
		t.Errorf("SRT:\n%s", formatSRT(cues))
	}
}

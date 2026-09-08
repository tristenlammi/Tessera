package subtitles

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Word-timed cues from whisper's full JSON output (-ojf) with DTW token timestamps
// (-dtw). Each token carries t_dtw, "the moment in audio in which the token was
// output" — in practice the token's end. A word ends at its last token's t_dtw and
// starts where the previous word ended. Cues are then built from words the way a
// subtitler would: broken at pauses and sentence ends, never past two lines, held on
// screen long enough to read. Segment start/end timestamps — the ones that put a cue
// up early after a pause — are only used as a bound.

type wsJSON struct {
	Transcription []wsSegment `json:"transcription"`
}

type wsSegment struct {
	Offsets wsOffsets `json:"offsets"`
	Text    string    `json:"text"`
	Tokens  []wsToken `json:"tokens"`
}

type wsToken struct {
	Text    string    `json:"text"`
	Offsets wsOffsets `json:"offsets"`
	TDTW    int64     `json:"t_dtw"`
	P       float64   `json:"p"`
}

type wsOffsets struct {
	From int64 `json:"from"` // ms
	To   int64 `json:"to"`   // ms
}

type word struct {
	start, end time.Duration
	text       string
}

const (
	wordMaxDur    = 800 * time.Millisecond  // bound on a first word's length when only its end is known
	cuePauseBreak = 1200 * time.Millisecond // a pause this long ends a cue
	cueSentGap    = 300 * time.Millisecond  // a sentence end plus this much pause ends a cue
	cueSentChars  = 40                      // ...once the cue has this much text
	cueEndPad     = 250 * time.Millisecond  // hold a cue a beat past its last word
	cueReadCPS    = 20                      // characters per second a viewer can read
)

// wordsFromJSON extracts timed words from whisper's JSON, adding base (the chunk's
// start) to every time. Segments whose DTW times are missing or don't make sense fall
// back to sharing the segment's time by character count.
func wordsFromJSON(data []byte, base time.Duration) ([]word, error) {
	var js wsJSON
	if err := json.Unmarshal(data, &js); err != nil {
		return nil, fmt.Errorf("whisper json: %w", err)
	}
	var out []word
	for _, seg := range js.Transcription {
		out = append(out, segmentWords(seg, base)...)
	}
	return out, nil
}

// isSpecialToken filters whisper's control tokens ("[_BEG_]", "[_TT_123]", "<|en|>").
func isSpecialToken(t string) bool {
	t = strings.TrimSpace(t)
	return t == "" || strings.HasPrefix(t, "[_") || strings.HasPrefix(t, "<|")
}

// segmentWords groups a segment's tokens into words (a token starting with a space
// starts a word; the rest attach) and times them.
func segmentWords(seg wsSegment, base time.Duration) []word {
	segFrom := base + time.Duration(seg.Offsets.From)*time.Millisecond
	segTo := base + time.Duration(seg.Offsets.To)*time.Millisecond
	type group struct {
		text    string
		lastTok int // index into seg.Tokens of the word's last token
	}
	var groups []group
	for i, t := range seg.Tokens {
		if isSpecialToken(t.Text) {
			continue
		}
		startsWord := strings.HasPrefix(t.Text, " ") || len(groups) == 0
		txt := strings.TrimSpace(t.Text)
		if txt == "" {
			continue
		}
		if startsWord {
			groups = append(groups, group{text: txt, lastTok: i})
		} else {
			groups[len(groups)-1].text += txt
			groups[len(groups)-1].lastTok = i
		}
	}
	if len(groups) == 0 {
		return nil
	}
	unit := dtwUnit(seg, segFrom-base, segTo-base)
	words := make([]word, 0, len(groups))
	if unit > 0 {
		prevEnd := segFrom
		for i, g := range groups {
			end := base + time.Duration(seg.Tokens[g.lastTok].TDTW)*unit
			start := prevEnd
			if i == 0 && end-start > wordMaxDur {
				// Only the end of the first word is known; whisper's segment start can
				// sit well before the speech, which is exactly the "too early" case.
				start = end - wordMaxDur
			}
			if end < start {
				end = start + 50*time.Millisecond
			}
			words = append(words, word{start: start, end: end, text: g.text})
			prevEnd = end
		}
		return words
	}
	// No usable DTW: share the segment's span by character count.
	total := 0
	for _, g := range groups {
		total += utf8.RuneCountInString(g.text) + 1
	}
	at := segFrom
	acc := 0
	for i, g := range groups {
		acc += utf8.RuneCountInString(g.text) + 1
		end := segFrom + time.Duration(float64(segTo-segFrom)*float64(acc)/float64(total))
		if i == len(groups)-1 {
			end = segTo
		}
		words = append(words, word{start: at, end: end, text: g.text})
		at = end
	}
	return words
}

// dtwUnit works out what t_dtw is counted in for this segment — centiseconds by the
// header's word, but checked against the segment's own millisecond offsets so a build
// that changes it can't silently shift every cue 10×. Returns 0 when the values are
// missing or don't land inside the segment (±2 s), which means "don't trust them".
func dtwUnit(seg wsSegment, from, to time.Duration) time.Duration {
	const slack = 2 * time.Second
	fits := func(unit time.Duration) bool {
		n := 0
		for _, t := range seg.Tokens {
			if isSpecialToken(t.Text) {
				continue
			}
			if t.TDTW < 0 {
				return false
			}
			at := time.Duration(t.TDTW) * unit
			if at < from-slack || at > to+slack {
				return false
			}
			n++
		}
		return n > 0
	}
	if fits(10 * time.Millisecond) {
		return 10 * time.Millisecond
	}
	if fits(time.Millisecond) {
		return time.Millisecond
	}
	return 0
}

// cuesFromWords packs timed words into cues: a new cue at a pause, at a sentence end
// once there's enough text, when the next word wouldn't fit on two lines, or when the
// cue would run too long.
func cuesFromWords(words []word) []cue {
	var out []cue
	var cur *cue
	var curChars int
	for _, w := range words {
		wl := utf8.RuneCountInString(w.text)
		if cur != nil {
			gap := w.start - cur.end
			brk := gap > cuePauseBreak ||
				curChars+1+wl > cueMaxChars ||
				w.end-cur.start > cueMaxDur ||
				(endsSentence(cur.text) && curChars >= cueSentChars && gap >= cueSentGap)
			if brk {
				out = append(out, *cur)
				cur = nil
			}
		}
		if cur == nil {
			c := cue{start: w.start, end: w.end, text: w.text}
			cur = &c
			curChars = wl
			continue
		}
		cur.text += " " + w.text
		cur.end = w.end
		curChars += 1 + wl
	}
	if cur != nil {
		out = append(out, *cur)
	}
	return out
}

func endsSentence(text string) bool {
	t := strings.TrimRight(text, `"')]`)
	return strings.HasSuffix(t, ".") || strings.HasSuffix(t, "!") || strings.HasSuffix(t, "?") || strings.HasSuffix(t, "…")
}

// padCues holds each cue a beat past its last word and long enough to read (at least
// a second, or the text's length at a comfortable reading speed), stopping short of
// the next cue.
func padCues(cues []cue) {
	for i := range cues {
		c := &cues[i]
		end := c.end + cueEndPad
		if read := c.start + time.Duration(float64(utf8.RuneCountInString(c.text))/cueReadCPS*float64(time.Second)); read > end {
			end = read
		}
		if end-c.start < cueMinDur {
			end = c.start + cueMinDur
		}
		if i+1 < len(cues) {
			if limit := cues[i+1].start - cueGuardGap; end > limit && limit > c.start {
				end = limit
			}
		}
		c.end = end
	}
}

// dropRepeats removes a hallucination loop: the same line three or more times in a
// row (whisper on music or crowd noise), keeping the first.
func dropRepeats(cues []cue) []cue {
	norm := func(s string) string {
		return strings.ToLower(strings.TrimSpace(strings.Trim(s, ".!?,…- ")))
	}
	out := make([]cue, 0, len(cues))
	run := 0
	for i, c := range cues {
		if i > 0 && norm(c.text) == norm(cues[i-1].text) {
			run++
		} else {
			run = 0
		}
		if run >= 2 {
			// Third identical line: this and every further repeat go; the first two
			// are dropped too since a doubled line is the loop's start.
			if run == 2 && len(out) >= 2 {
				out = out[:len(out)-1]
			}
			continue
		}
		out = append(out, c)
	}
	return out
}

// shapeWordCues is the whole pipeline from timed words to laid-out cues.
func shapeWordCues(words []word) []cue {
	cues := cuesFromWords(words)
	cues = mergeFragments(cues)
	cues = dropRepeats(cues)
	padCues(cues)
	for i := range cues {
		cues[i].text = layoutLines(cues[i].text)
	}
	return cues
}

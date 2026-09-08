package subtitles

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Cue shaping for AI-generated subtitles: the SRT model, the limits, and the passes
// shared by the word-timed pipeline (words.go) — fragment merging, duration floors,
// two-line layout. splitCue is the character-share fallback for a segment whose word
// times are unusable.
const (
	cueMaxChars   = 84 // two lines of 42 — the usual broadcast limit
	cueLineChars  = 42
	cueMinDur     = time.Second
	cueMaxDur     = 7 * time.Second
	cueMergeGap   = 400 * time.Millisecond // neighbours closer than this are one utterance
	cueShortChars = 25                     // a cue shorter than this is a fragment
	cueGuardGap   = 50 * time.Millisecond  // keep between a cue's end and the next start
)

type cue struct {
	start, end time.Duration
	text       string // single line, whitespace collapsed
}

var srtTimeRe = regexp.MustCompile(`(\d+):(\d\d):(\d\d)[,.](\d{1,3})\s*-->\s*(\d+):(\d\d):(\d\d)[,.](\d{1,3})`)

// parseSRT reads cues from SRT text; blocks without a timing line are skipped.
func parseSRT(s string) []cue {
	var out []cue
	for _, block := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n\n") {
		var c cue
		var text []string
		timed := false
		for _, ln := range strings.Split(block, "\n") {
			t := strings.TrimSpace(ln)
			if t == "" {
				continue
			}
			if m := srtTimeRe.FindStringSubmatch(t); m != nil && !timed {
				c.start, c.end = srtTime(m[1:5]), srtTime(m[5:9])
				timed = true
				continue
			}
			if !timed && srtIndexLine.MatchString(t) {
				continue
			}
			if timed {
				text = append(text, t)
			}
		}
		if !timed {
			continue
		}
		c.text = strings.Join(strings.Fields(strings.Join(text, " ")), " ")
		if c.text == "" {
			continue
		}
		out = append(out, c)
	}
	return out
}

func srtTime(p []string) time.Duration {
	h, _ := strconv.Atoi(p[0])
	m, _ := strconv.Atoi(p[1])
	s, _ := strconv.Atoi(p[2])
	ms, _ := strconv.Atoi((p[3] + "00")[:3]) // "5" → 500ms, "50" → 500ms, "500" → 500ms
	return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(s)*time.Second + time.Duration(ms)*time.Millisecond
}

func fmtSRTTime(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	ms := d.Milliseconds()
	return fmt.Sprintf("%02d:%02d:%02d,%03d", ms/3600000, ms/60000%60, ms/1000%60, ms%1000)
}

func formatSRT(cues []cue) string {
	var b strings.Builder
	for i, c := range cues {
		b.WriteString(strconv.Itoa(i + 1))
		b.WriteString("\n")
		b.WriteString(fmtSRTTime(c.start))
		b.WriteString(" --> ")
		b.WriteString(fmtSRTTime(c.end))
		b.WriteString("\n")
		b.WriteString(c.text)
		b.WriteString("\n\n")
	}
	return b.String()
}

// splitCue breaks a cue that is too long (in characters or seconds) into pieces, cutting
// at punctuation where it can, and gives each piece a share of the time in proportion
// to its length — a fair stand-in for word timing when the segment is one sentence.
func splitCue(c cue) []cue {
	words := strings.Fields(c.text)
	chars := utf8.RuneCountInString(c.text)
	dur := c.end - c.start
	pieces := int(math.Ceil(float64(chars) / cueMaxChars))
	if dur > cueMaxDur && len(words) >= 6 {
		if byTime := int(math.Ceil(float64(dur) / float64(cueMaxDur))); byTime > pieces {
			pieces = byTime
		}
	}
	if pieces <= 1 || len(words) < 2 {
		return []cue{c}
	}
	budget := int(math.Ceil(float64(chars) / float64(pieces)))
	if budget > cueMaxChars {
		budget = cueMaxChars
	}
	chunks := chunkWords(words, budget)
	if len(chunks) == 1 {
		return []cue{c}
	}
	// Time by character share, contiguous, ending exactly where the segment did.
	total := 0
	for _, ch := range chunks {
		total += utf8.RuneCountInString(ch)
	}
	out := make([]cue, 0, len(chunks))
	at := c.start
	acc := 0
	for i, ch := range chunks {
		acc += utf8.RuneCountInString(ch)
		end := c.start + time.Duration(float64(dur)*float64(acc)/float64(total))
		if i == len(chunks)-1 {
			end = c.end
		}
		out = append(out, cue{start: at, end: end, text: ch})
		at = end
	}
	return out
}

// chunkWords packs words into chunks of at most budget characters. When a chunk has to
// close, it closes at the last clause boundary (. , ! ? ; :) in its second half if there
// is one, so pieces read as phrases rather than arbitrary runs of words.
func chunkWords(words []string, budget int) []string {
	var chunks []string
	var cur []string
	curLen := 0
	lastPunct := -1 // index in cur of the last word ending in punctuation
	flush := func(upto int) {
		chunks = append(chunks, strings.Join(cur[:upto], " "))
		rest := append([]string(nil), cur[upto:]...)
		cur = rest
		curLen = 0
		lastPunct = -1
		for i, w := range cur {
			curLen += utf8.RuneCountInString(w) + 1
			if endsClause(w) {
				lastPunct = i
			}
		}
		if curLen > 0 {
			curLen--
		}
	}
	for _, w := range words {
		wl := utf8.RuneCountInString(w)
		add := wl
		if curLen > 0 {
			add++
		}
		if curLen+add > budget && len(cur) > 0 {
			// Close the chunk: at the clause boundary if it sits in the second half.
			if lastPunct >= 0 && lastPunct+1 < len(cur) && runeLen(cur[:lastPunct+1]) >= budget*2/5 {
				flush(lastPunct + 1)
			} else {
				flush(len(cur))
			}
			add = wl
			if curLen > 0 {
				add = wl + 1
			}
		}
		cur = append(cur, w)
		curLen += add
		if endsClause(w) {
			lastPunct = len(cur) - 1
		}
	}
	if len(cur) > 0 {
		chunks = append(chunks, strings.Join(cur, " "))
	}
	return chunks
}

func runeLen(words []string) int {
	n := 0
	for _, w := range words {
		n += utf8.RuneCountInString(w) + 1
	}
	if n > 0 {
		n--
	}
	return n
}

func endsClause(w string) bool {
	w = strings.TrimRight(w, `"')]`)
	return strings.HasSuffix(w, ".") || strings.HasSuffix(w, ",") || strings.HasSuffix(w, "!") ||
		strings.HasSuffix(w, "?") || strings.HasSuffix(w, ";") || strings.HasSuffix(w, ":") || strings.HasSuffix(w, "…")
}

// mergeFragments joins a fragment onto the neighbour it's contiguous with — forward
// first, then back — provided the result still fits on two lines.
func mergeFragments(cues []cue) []cue {
	if len(cues) < 2 {
		return cues
	}
	out := make([]cue, 0, len(cues))
	for i := 0; i < len(cues); i++ {
		c := cues[i]
		if utf8.RuneCountInString(c.text) >= cueShortChars {
			out = append(out, c)
			continue
		}
		if i+1 < len(cues) && canMerge(c, cues[i+1]) {
			cues[i+1] = cue{start: c.start, end: cues[i+1].end, text: c.text + " " + cues[i+1].text}
			continue
		}
		if len(out) > 0 && canMerge(out[len(out)-1], c) {
			p := &out[len(out)-1]
			p.text += " " + c.text
			p.end = c.end
			continue
		}
		out = append(out, c)
	}
	return out
}

func canMerge(a, b cue) bool {
	gap := b.start - a.end
	if gap < 0 {
		gap = 0
	}
	return gap <= cueMergeGap &&
		utf8.RuneCountInString(a.text)+1+utf8.RuneCountInString(b.text) <= cueMaxChars &&
		b.end-a.start <= cueMaxDur+cueMergeGap
}

// floorDurations gives every cue at least cueMinDur on screen, stopping short of the
// next cue, and removes any overlap whisper left behind.
func floorDurations(cues []cue) {
	for i := range cues {
		c := &cues[i]
		if c.end-c.start < cueMinDur {
			c.end = c.start + cueMinDur
		}
		if i+1 < len(cues) {
			if limit := cues[i+1].start - cueGuardGap; c.end > limit && limit > c.start {
				c.end = limit
			}
		}
	}
}

// layoutLines breaks text over one line into two, at the word boundary nearest the
// middle — preferring one after punctuation — so both lines read as phrases.
func layoutLines(text string) string {
	if utf8.RuneCountInString(text) <= cueLineChars {
		return text
	}
	words := strings.Fields(text)
	if len(words) < 2 {
		return text
	}
	total := utf8.RuneCountInString(text)
	best, bestScore := -1, math.MaxFloat64
	left := 0
	for i := 0; i < len(words)-1; i++ {
		left += utf8.RuneCountInString(words[i])
		if i > 0 {
			left++
		}
		right := total - left - 1
		if left > cueLineChars || right > cueLineChars {
			continue
		}
		score := math.Abs(float64(left - right))
		if endsClause(words[i]) {
			score -= 8 // a clause break is worth a lopsided split
		}
		if score < bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		// Nothing fits in 42/42 — break as evenly as words allow.
		left = 0
		for i := 0; i < len(words)-1; i++ {
			left += utf8.RuneCountInString(words[i]) + 1
			if left >= total/2 {
				best = i
				break
			}
		}
		if best < 0 {
			return text
		}
	}
	return strings.Join(words[:best+1], " ") + "\n" + strings.Join(words[best+1:], " ")
}

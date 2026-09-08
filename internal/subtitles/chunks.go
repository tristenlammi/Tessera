package subtitles

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// Chunked transcription.
//
// whisper runs on stretches of speech cut at silences, rather than on the whole file
// with its built-in VAD. Same goal — music and silence aren't transcribed, so they
// aren't hallucinated over — but every timestamp whisper reports is on a continuous
// timeline that only needs the chunk's start added. The VAD path instead removes the
// silences, transcribes what's left, and maps timestamps back afterwards; a decoded
// segment that straddled a removed silence came back stamped at the earlier speech
// chunk's start, which is the "subtitle appears way too early" the user saw.
//
// Chunks are a few minutes long (one model load each — a few seconds on the GPU) and
// end inside a pause, so no word is cut in half.
const (
	chunkSilenceMin = 1200 * time.Millisecond // a pause this long may separate chunks
	chunkSilencePad = 300 * time.Millisecond  // keep this much of the pause on each side
	chunkTarget     = 4 * time.Minute         // cut at the first qualifying pause past this
	chunkMax        = 8 * time.Minute         // ...and at the last usable pause before this
	chunkNoise      = "-35dB"                 // silencedetect threshold; dialogue sits well above
)

type chunk struct{ start, end time.Duration }

// silence is one detected gap in the audio.
type silence struct{ start, end time.Duration }

var silenceLineRe = regexp.MustCompile(`silence_(start|end):\s*(-?\d+(?:\.\d+)?)`)

// detectSilences runs ffmpeg's silencedetect over the extracted WAV. total bounds a
// trailing silence that has no end line (the file ended inside it).
func detectSilences(ctx context.Context, ffmpeg, wav string, total time.Duration) ([]silence, error) {
	out, err := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-nostats", "-i", wav,
		"-af", fmt.Sprintf("silencedetect=noise=%s:d=%.2f", chunkNoise, chunkSilenceMin.Seconds()),
		"-f", "null", "-").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("silencedetect: %w: %s", err, tailStr(out, 200))
	}
	return parseSilences(string(out), total), nil
}

// parseSilences pairs silencedetect's start/end lines in order.
func parseSilences(out string, total time.Duration) []silence {
	var sils []silence
	open := time.Duration(-1)
	for _, m := range silenceLineRe.FindAllStringSubmatch(out, -1) {
		sec, err := strconv.ParseFloat(m[2], 64)
		if err != nil {
			continue
		}
		at := time.Duration(sec * float64(time.Second))
		if at < 0 {
			at = 0
		}
		switch m[1] {
		case "start":
			open = at
		case "end":
			if open >= 0 && at > open {
				sils = append(sils, silence{open, at})
			}
			open = -1
		}
	}
	if open >= 0 && total > open {
		sils = append(sils, silence{open, total}) // ran off the end of the file
	}
	sort.Slice(sils, func(i, j int) bool { return sils[i].start < sils[j].start })
	return sils
}

// planChunks turns the silences into chunk boundaries. Each chunk ends a little way
// into a pause and the next starts a little way before the pause ends, so the pause
// itself (and any long stretch of music or nothing) is mostly left out. A file with no
// usable pauses is one chunk.
func planChunks(total time.Duration, sils []silence) []chunk {
	if total <= 0 {
		return nil
	}
	var out []chunk
	start := time.Duration(0)
	lastGood := time.Duration(-1) // a pause past half the target, usable if we overrun
	lastGoodEnd := time.Duration(0)
	emit := func(end, next time.Duration) {
		if end-start >= time.Second {
			out = append(out, chunk{start, end})
		}
		start = next
		lastGood = -1
	}
	for _, s := range sils {
		if s.end-s.start < chunkSilenceMin {
			continue
		}
		cutEnd := s.start + chunkSilencePad // this chunk ends here
		cutNext := s.end - chunkSilencePad  // the next starts here
		if s.start == 0 {
			// Leading silence: nothing to transcribe before it.
			start = cutNext
			continue
		}
		if cutEnd <= start {
			continue
		}
		if cutEnd-start > chunkMax && lastGood > start {
			// Overran the maximum without a pause past the target: cut at the last decent
			// pause, then reconsider this one against the new start.
			end, next := lastGood, lastGoodEnd
			emit(end, next)
		}
		if cutEnd-start >= chunkTarget {
			emit(cutEnd, cutNext)
			continue
		}
		if cutEnd-start >= chunkTarget/2 {
			lastGood, lastGoodEnd = cutEnd, cutNext
		}
	}
	if total > start && total-start >= time.Second {
		out = append(out, chunk{start, total})
	}
	if len(out) == 0 {
		return []chunk{{0, total}}
	}
	return out
}

// cutChunk writes one chunk of the WAV to dst (PCM copy — no re-encode, sample-exact).
func cutChunk(ctx context.Context, ffmpeg, wav string, c chunk, dst string) error {
	out, err := exec.CommandContext(ctx, ffmpeg, "-y", "-hide_banner", "-nostats",
		"-ss", fmt.Sprintf("%.3f", c.start.Seconds()), "-t", fmt.Sprintf("%.3f", (c.end-c.start).Seconds()),
		"-i", wav, "-c", "copy", dst).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cut chunk: %w: %s", err, tailStr(out, 200))
	}
	if fi, err := os.Stat(dst); err != nil || fi.Size() < 1024 {
		return fmt.Errorf("cut chunk: empty output")
	}
	return nil
}

// wavDuration derives the length of a 16 kHz mono 16-bit WAV from its size (the
// header is a few dozen bytes; the error is well under a millisecond).
func wavDuration(path string) time.Duration {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() <= 44 {
		return 0
	}
	return time.Duration(float64(fi.Size()-44) / 32000 * float64(time.Second))
}

// overallProgress maps one chunk's 0-100 onto the whole file's.
func overallProgress(c chunk, pct int, total time.Duration) int {
	if total <= 0 {
		return pct
	}
	at := c.start + time.Duration(float64(c.end-c.start)*float64(pct)/100)
	p := int(float64(at) / float64(total) * 100)
	if p < 0 {
		p = 0
	}
	if p > 100 {
		p = 100
	}
	return p
}

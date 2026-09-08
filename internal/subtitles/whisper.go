package subtitles

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

// whisperGen wraps the bundled whisper.cpp CLI for local AI subtitle generation. It's inert when
// the binary or a model isn't present — the module then reports AI as unavailable instead of
// failing, so everything builds/runs before the Dockerfile bundles whisper.
type whisperGen struct {
	bin          string // whisper-cli path ("" = not installed)
	modelsDir    string // where the GGML model files live (data dir / whisper)
	vadSupported bool   // whether this whisper-cli build understands --vad
	noGPUFlag    bool   // whether this build understands --no-gpu (i.e. was built with a GPU backend)
	noFallback   bool   // --no-fallback available
	suppressNST  bool   // --suppress-nst available
	dlMu         sync.Mutex
	dl           map[string]bool // model filenames currently downloading

	// backend is what the last run actually used: "vulkan", "cpu", or "" before any run.
	// Learned from whisper-cli's own output rather than guessed from the build, since a
	// Vulkan build on a host with no visible device silently runs on the CPU.
	backendMu sync.Mutex
	backend   string
}

// Backend reports which compute backend the AI last ran on, for the status panel.
func (w *whisperGen) Backend() string {
	if w == nil {
		return ""
	}
	w.backendMu.Lock()
	defer w.backendMu.Unlock()
	return w.backend
}

func (w *whisperGen) setBackend(b string) {
	w.backendMu.Lock()
	w.backend = b
	w.backendMu.Unlock()
}

// threads is how many CPU threads to give whisper-cli. Its default is 4, which on a
// 24-core host left five-sixths of the machine idle and made a 45-minute episode take
// most of an hour. Diminishing returns past ~16 on the decoder, so cap there.
func threads() int {
	n := runtime.NumCPU()
	if n > 16 {
		n = 16
	}
	if n < 1 {
		n = 1
	}
	return n
}

// backendOf reads which backend whisper-cli reported using. The Vulkan build prints a
// "ggml_vulkan: Found N Vulkan devices" line at startup when it has a device; without one
// it says nothing about Vulkan and runs on the CPU.
func backendOf(out []byte) string {
	if bytes.Contains(out, []byte("ggml_vulkan: Found")) || bytes.Contains(out, []byte("Vulkan0")) {
		return "vulkan"
	}
	return "cpu"
}

// GGML model + VAD filenames (from ggerganov/whisper.cpp on Hugging Face). turbo is fast and used
// for same-language transcription; large-v3 is required for translate-to-English (turbo can't
// translate). silero is the VAD model that suppresses non-speech hallucination.
const (
	modelTurbo = "ggml-large-v3-turbo.bin"
	modelLarge = "ggml-large-v3.bin"
	vadModel   = "ggml-silero-v6.2.0.bin"
)

func detectWhisper(modelsDir string) *whisperGen {
	bin, _ := exec.LookPath("whisper-cli")
	w := &whisperGen{bin: bin, modelsDir: modelsDir, dl: map[string]bool{}}
	if bin != "" {
		// Older whisper.cpp builds don't have VAD; passing --vad makes them print usage and do
		// nothing. Probe the help text once so we only pass it when supported.
		out, _ := exec.Command(bin, "--help").CombinedOutput()
		w.vadSupported = strings.Contains(string(out), "--vad")
		w.noGPUFlag = strings.Contains(string(out), "--no-gpu")
		w.noFallback = strings.Contains(string(out), "--no-fallback")
		w.suppressNST = strings.Contains(string(out), "--suppress-nst")
	}
	return w
}

// vadActive reports whether runs are VAD-fronted: the build supports it and the model is here.
func (w *whisperGen) vadActive() bool {
	return w != nil && w.vadSupported && w.hasModel(vadModel)
}

func (w *whisperGen) hasModel(name string) bool {
	if w == nil || w.modelsDir == "" {
		return false
	}
	fi, err := os.Stat(filepath.Join(w.modelsDir, name))
	return err == nil && fi.Size() > 0
}

// available reports whether generation can actually run (binary + at least one usable model).
func (w *whisperGen) available() bool {
	return w != nil && w.bin != "" && (w.hasModel(modelTurbo) || w.hasModel(modelLarge))
}

// modelPath picks the model for the task: translate-to-English requires large-v3 (turbo can't
// translate); same-language transcription prefers turbo and falls back to large-v3. "" = no
// suitable model.
func (w *whisperGen) modelPath(translate bool) string {
	if translate {
		if w.hasModel(modelLarge) {
			return filepath.Join(w.modelsDir, modelLarge)
		}
		return ""
	}
	if w.hasModel(modelTurbo) {
		return filepath.Join(w.modelsDir, modelTurbo)
	}
	if w.hasModel(modelLarge) {
		return filepath.Join(w.modelsDir, modelLarge)
	}
	return ""
}

// generate produces an SRT for one language from a video's audio: extract 16 kHz mono → run
// whisper.cpp (VAD-fronted when the VAD model is present) → the SRT sidecar, then strip stock-phrase
// hallucinations. translate=true asks Whisper to translate the (foreign) audio to English.
func (w *whisperGen) generate(ctx context.Context, ffmpeg, videoPath, srtPath, lang string, translate bool, progress func(pct int)) error {
	model := w.modelPath(translate)
	if model == "" {
		return fmt.Errorf("no whisper model available for %s", ifElse(translate, "translation", "transcription"))
	}
	// Work in the temp dir under a clean (space-free) base, then move the result next to the video.
	// Whisper writes "<outBase>.srt".
	stamp := time.Now().UnixNano()
	wav := filepath.Join(os.TempDir(), fmt.Sprintf("whisper-%d.wav", stamp))
	outBase := filepath.Join(os.TempDir(), fmt.Sprintf("whisper-%d", stamp))
	outSRT := outBase + ".srt"
	defer os.Remove(wav)
	defer os.Remove(outSRT)

	// 1. Extract mono 16 kHz PCM — what Whisper expects — and confirm it's real audio.
	if out, err := exec.CommandContext(ctx, ffmpeg, "-y", "-hide_banner", "-i", videoPath,
		"-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", wav).CombinedOutput(); err != nil {
		return fmt.Errorf("extract audio: %w: %s", err, tailStr(out, 300))
	}
	if fi, err := os.Stat(wav); err != nil || fi.Size() < 4096 {
		return fmt.Errorf("extracted audio was empty (no decodable audio track?)")
	}

	// 2. Transcribe/translate to SRT.
	//
	// Plain segments, deliberately: -ml/-sow (split long segments in whisper) run on its
	// experimental token-level timestamps, which put three words on screen for a moment
	// each, out of step with the audio. Segment timestamps are the reliable ones; the
	// cues are shaped afterwards in shapeCues (cues.go).
	//
	// -pp prints "progress = N%" lines, which runWhisper turns into the job's progress.
	args := []string{"-m", model, "-f", wav, "-osrt", "-of", outBase, "-t", strconv.Itoa(threads()), "-pp"}
	if translate {
		args = append(args, "--translate")
	} else if lang != "" {
		args = append(args, "-l", lang)
	}
	if w.vadActive() {
		args = append(args, "--vad", "--vad-model", filepath.Join(w.modelsDir, vadModel))
	}
	// Temperature fallback re-decodes a window up to five more times when the first
	// decode looks poor. The windows that look poor are music, silence and crowd noise,
	// and every retry there produces the same invented line — the "I'm not saying this is
	// how I'm going to be" once a second in the log — so it multiplies the run time for
	// nothing. VAD is the right tool for those stretches; the fallback is switched off.
	// Non-speech tokens (♪, [Music]) are suppressed for the same reason.
	if w.noFallback {
		args = append(args, "-nf")
	}
	if w.suppressNST {
		args = append(args, "--suppress-nst")
	}
	out, err := runWhisper(ctx, w.bin, args, progress)
	if err != nil && w.noGPUFlag && ctx.Err() == nil {
		// A GPU build that can't initialise its device (driver missing in the container,
		// /dev/dri not passed through, an out-of-memory on a small card) fails outright
		// rather than degrading. Retry on the CPU: slower, but it produces the subtitle.
		gpuErr := tailStr(out, 300)
		out, err = runWhisper(ctx, w.bin, append(args, "--no-gpu"), progress)
		if err == nil {
			w.setBackend("cpu")
			return w.finishSRT(outSRT, srtPath, out, "whisper ran on the CPU after the GPU attempt failed: "+gpuErr)
		}
	}
	if err != nil {
		return fmt.Errorf("whisper: %w: %s", err, tailStr(out, 400))
	}
	w.setBackend(backendOf(out))
	return w.finishSRT(outSRT, srtPath, out, "")
}

// finishSRT checks whisper actually wrote something, strips stock-phrase hallucinations,
// and writes the sidecar next to the video.
func (w *whisperGen) finishSRT(outSRT, srtPath string, out []byte, note string) error {
	_ = note // surfaced by the caller's event log via Backend(); kept for the error path
	if fi, statErr := os.Stat(outSRT); statErr != nil || fi.Size() == 0 {
		return fmt.Errorf("whisper produced no subtitle output: %s", tailStr(out, 400))
	}

	// 3. Filter stock-phrase hallucinations and write the sidecar next to the video.
	b, err := os.ReadFile(outSRT)
	if err != nil {
		return err
	}
	return os.WriteFile(srtPath, []byte(shapeCues(filterStockPhrases(string(b)))), 0o644)
}

// progressRe matches whisper-cli's -pp output: "whisper_print_progress_callback: progress =  25%".
var progressRe = regexp.MustCompile(`progress\s*=\s*(\d{1,3})%`)

// runWhisper runs whisper-cli, streaming its output so progress lines reach the caller
// while it runs, and returns the full combined output for diagnostics afterwards. The old
// CombinedOutput held everything until exit — fine for errors, useless for a bar.
func runWhisper(ctx context.Context, bin string, args []string, progress func(int)) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	pr, pw := io.Pipe()
	cmd.Stdout = pw
	cmd.Stderr = pw
	if err := cmd.Start(); err != nil {
		pw.Close()
		return nil, err
	}
	var buf bytes.Buffer
	done := make(chan struct{})
	go func() {
		defer close(done)
		sc := bufio.NewScanner(pr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		last := -1
		for sc.Scan() {
			line := sc.Bytes()
			buf.Write(line)
			buf.WriteByte('\n')
			if progress == nil {
				continue
			}
			if m := progressRe.FindSubmatch(line); m != nil {
				if pct, err := strconv.Atoi(string(m[1])); err == nil && pct != last && pct >= 0 && pct <= 100 {
					last = pct
					progress(pct)
				}
			}
		}
	}()
	err := cmd.Wait()
	pw.Close()
	<-done
	return buf.Bytes(), err
}

// tailStr returns the last n bytes of out as a single trimmed line (for error diagnostics).
func tailStr(out []byte, n int) string {
	s := strings.TrimSpace(string(out))
	if len(s) > n {
		s = s[len(s)-n:]
	}
	return strings.ReplaceAll(s, "\n", " ⏎ ")
}

// aiPlan decides how (if at all) AI can produce a wanted-language subtitle from a file's audio:
//
//	"transcribe" — the audio is already in that language;
//	"translate"  — the target is English and the audio isn't (Whisper's only translation direction),
//	               or the audio language is unknown (auto-detect + translate is a safe default);
//	""           — impossible: Whisper can't translate into a non-English language.
func aiPlan(audioLangs []string, wanted string) string {
	for _, a := range audioLangs {
		if langMatches(a, wanted) {
			return "transcribe"
		}
	}
	if isEnglish(wanted) {
		return "translate"
	}
	return ""
}

func isEnglish(l string) bool {
	l = strings.ToLower(strings.TrimSpace(l))
	return l == "en" || l == "eng" || l == "english"
}

func ifElse(c bool, a, b string) string {
	if c {
		return a
	}
	return b
}

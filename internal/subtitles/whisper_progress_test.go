package subtitles

import "testing"

// whisper-cli's -pp output is the only progress signal there is; the bar depends on
// reading it exactly as printed, padding and all.
func TestProgressLineParsing(t *testing.T) {
	for line, want := range map[string]int{
		"whisper_print_progress_callback: progress =   5%": 5,
		"whisper_print_progress_callback: progress =  25%": 25,
		"whisper_print_progress_callback: progress = 100%": 100,
		"whisper_print_progress_callback: progress =   0%": 0,
	} {
		m := progressRe.FindStringSubmatch(line)
		if m == nil {
			t.Errorf("%q: no match", line)
			continue
		}
		if got := m[1]; got != itoa(want) {
			t.Errorf("%q: parsed %q, want %d", line, got, want)
		}
	}
	// Ordinary output must not be mistaken for progress.
	for _, line := range []string{
		"system_info: n_threads = 16 / 24 | AVX = 1",
		"whisper_init_from_file_with_params_no_state: loading model",
		"[00:00:05.000 --> 00:00:08.500]   I said 100% of the time.",
	} {
		if progressRe.MatchString(line) {
			t.Errorf("%q matched as a progress line", line)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d [4]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	return string(d[i:])
}

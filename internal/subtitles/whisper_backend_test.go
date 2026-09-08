package subtitles

import (
	"runtime"
	"testing"
)

// The Vulkan build announces its device at startup; a build with no visible device says
// nothing about Vulkan and runs on the CPU. The status panel must tell those apart, since
// "AI ready" on a GPU build that's silently on the CPU is the most likely misconfiguration.
func TestBackendOfReadsWhisperOutput(t *testing.T) {
	for name, tc := range map[string]struct {
		out  string
		want string
	}{
		"vulkan device found": {"ggml_vulkan: Found 1 Vulkan devices:\nVulkan0: Intel(R) Arc(tm) A380 Graphics (DG2) | uma: 0 | fp16: 1\nwhisper_init_from_file_with_params_no_state: loading model", "vulkan"},
		"cpu only build":      {"whisper_init_from_file_with_params_no_state: loading model from 'ggml-large-v3-turbo.bin'\nsystem_info: n_threads = 16", "cpu"},
		"empty":               {"", "cpu"},
	} {
		if got := backendOf([]byte(tc.out)); got != tc.want {
			t.Errorf("%s: backendOf = %q, want %q", name, got, tc.want)
		}
	}
}

// whisper-cli's own default is 4 threads. Passing the host's core count — capped where
// the decoder stops scaling — is the cheapest speedup available.
func TestThreadsUsesTheMachine(t *testing.T) {
	n := threads()
	if n < 1 {
		t.Fatalf("threads() = %d", n)
	}
	if n > 16 {
		t.Errorf("threads() = %d, want capped at 16", n)
	}
	if cpus := runtime.NumCPU(); cpus <= 16 && n != cpus {
		t.Errorf("threads() = %d on a %d-core machine, want all of them", n, cpus)
	}
}

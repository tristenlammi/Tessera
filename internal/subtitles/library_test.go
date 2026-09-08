package subtitles

import "testing"

func TestLangMatches(t *testing.T) {
	cases := []struct {
		track, wanted string
		want          bool
	}{
		{"eng", "en", true},
		{"en", "en", true},
		{"eng", "eng", true},
		{"spa", "es", true},
		{"fre", "fr", true},
		{"eng", "es", false},
		{"", "en", false},    // untagged track never matches a specific language
		{"und", "en", false}, // undefined ditto
		{"en", "", false},
	}
	for _, c := range cases {
		if got := langMatches(c.track, c.wanted); got != c.want {
			t.Errorf("langMatches(%q,%q) = %v, want %v", c.track, c.wanted, got, c.want)
		}
	}
}

func TestBestSource(t *testing.T) {
	textEn := SubTrack{Codec: "subrip", Lang: "eng", Text: true}
	pgsEn := SubTrack{Codec: "hdmv_pgs_subtitle", Lang: "eng", Text: false}
	textFr := SubTrack{Codec: "ass", Lang: "fre", Text: true}

	// Embedded English text → extract (highest priority).
	if got := bestSource([]SubTrack{textEn, pgsEn}, "en", true); got != "extract" {
		t.Errorf("with embedded text, source = %q, want extract", got)
	}
	// Only an English image sub. OCR would be the next-best source, but it isn't
	// implemented, and routing there sent every PGS-only file to "pending" without ever
	// trying a download — the file that most needs a text subtitle, since an image track
	// is what makes Plex burn subtitles in and transcode. Until OCR exists, fall through.
	if got := bestSource([]SubTrack{pgsEn}, "en", true); got != "download" {
		t.Errorf("with image sub only + provider, source = %q, want download", got)
	}
	if got := bestSource([]SubTrack{pgsEn}, "en", false); got != "ai" {
		t.Errorf("with image sub only + no provider, source = %q, want ai", got)
	}
	// No English track, but the provider can download → download.
	if got := bestSource([]SubTrack{textFr}, "en", true); got != "download" {
		t.Errorf("with only other-lang track + provider, source = %q, want download", got)
	}
	// Nothing usable and no provider → AI fallback.
	if got := bestSource([]SubTrack{textFr}, "en", false); got != "ai" {
		t.Errorf("with nothing + no provider, source = %q, want ai", got)
	}
	// The French text track should extract for French even alongside English subs.
	if got := bestSource([]SubTrack{textEn, textFr}, "fr", false); got != "extract" {
		t.Errorf("for fr, source = %q, want extract", got)
	}
}

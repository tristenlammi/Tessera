package convert

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The goal this exists for: a file whose only English subtitle is PGS makes Plex burn it
// in and transcode the video. Once a text English subtitle exists, the PGS track can go
// and the file direct-plays.
func TestDropImageSubsOnlyWhenTextExists(t *testing.T) {
	pgsEn := SubStream{SubIndex: 0, Codec: "hdmv_pgs_subtitle", Lang: "eng", Text: false}
	srtEn := SubStream{SubIndex: 1, Codec: "subrip", Lang: "eng", Text: true}
	pgsFr := SubStream{SubIndex: 2, Codec: "hdmv_pgs_subtitle", Lang: "fre", Text: false}
	drop := Plan{Subs: SubPlan{DropImage: true}}

	// Embedded text English alongside PGS English: the PGS goes, the text stays.
	got := keptSubs(&MediaInfo{Subs: []SubStream{pgsEn, srtEn}}, drop)
	if len(got) != 1 || !got[0].Text {
		t.Errorf("with an embedded text track, kept %+v; want just the text track", got)
	}

	// PGS English only, nothing else: it is the only subtitle there is. It stays until
	// the Subtitles module has produced a text one.
	got = keptSubs(&MediaInfo{Subs: []SubStream{pgsEn}}, drop)
	if len(got) != 1 {
		t.Errorf("PGS-only file lost its only subtitle: kept %+v", got)
	}

	// PGS English only, but an .srt sidecar exists beside the file: now it can go.
	withSidecar := drop
	withSidecar.Subs.TextSidecarLangs = []string{"en"}
	got = keptSubs(&MediaInfo{Subs: []SubStream{pgsEn}}, withSidecar)
	if len(got) != 0 {
		t.Errorf("PGS English with an English .srt beside it was kept: %+v", got)
	}

	// The sidecar is English; the French PGS has no text alternative and stays.
	got = keptSubs(&MediaInfo{Subs: []SubStream{pgsEn, pgsFr}}, withSidecar)
	if len(got) != 1 || got[0].Lang != "fre" {
		t.Errorf("kept %+v; want only the French PGS (no French text exists)", got)
	}

	// Off: nothing image-related happens at all.
	got = keptSubs(&MediaInfo{Subs: []SubStream{pgsEn, srtEn}}, Plan{})
	if len(got) != 2 {
		t.Errorf("with DropImage off, kept %d of 2", len(got))
	}
}

// Language filter and image filter compose: keep English, drop images. A French text
// track goes on language; the English PGS goes on type because English text remains.
func TestDropImageComposesWithLanguageFilter(t *testing.T) {
	mi := &MediaInfo{Subs: []SubStream{
		{SubIndex: 0, Codec: "hdmv_pgs_subtitle", Lang: "eng", Text: false},
		{SubIndex: 1, Codec: "subrip", Lang: "eng", Text: true},
		{SubIndex: 2, Codec: "subrip", Lang: "fre", Text: true},
	}}
	got := keptSubs(mi, Plan{Subs: SubPlan{KeepLangs: []string{"en"}, DropImage: true}})
	if len(got) != 1 || got[0].SubIndex != 1 {
		t.Errorf("kept %+v; want only the English text track", got)
	}
}

// An untagged image track can't be matched by language, so it's dropped only when SOME
// text subtitle exists — the viewer is never left with nothing.
func TestUntaggedImageTrackNeedsAnyText(t *testing.T) {
	untagged := SubStream{SubIndex: 0, Codec: "dvd_subtitle", Lang: "", Text: false}
	drop := Plan{Subs: SubPlan{DropImage: true}}
	if got := keptSubs(&MediaInfo{Subs: []SubStream{untagged}}, drop); len(got) != 1 {
		t.Errorf("an untagged image track that is the only subtitle was dropped")
	}
	withSidecar := drop
	withSidecar.Subs.TextSidecarLangs = []string{"en"}
	if got := keptSubs(&MediaInfo{Subs: []SubStream{untagged}}, withSidecar); len(got) != 0 {
		t.Errorf("an untagged image track was kept despite a text sidecar: %+v", got)
	}
}

// Needs must report a PGS-only file as work to do ONLY once its sidecar exists —
// otherwise the Library would list files Convert can't actually change yet.
func TestNeedsReflectsSidecarAvailability(t *testing.T) {
	mi := &MediaInfo{VideoCodec: "hevc", Subs: []SubStream{
		{SubIndex: 0, Codec: "hdmv_pgs_subtitle", Lang: "eng", Text: false},
	}}
	dp := Plan{Subs: SubPlan{DropImage: true}}
	if n := needsOf(mi, dp, "hevc", false); n.Subs {
		t.Error("PGS-only file with no text subtitle was listed as fixable — nothing can be dropped yet")
	}
	dp.Subs.TextSidecarLangs = []string{"en"}
	if n := needsOf(mi, dp, "hevc", false); !n.Subs {
		t.Error("PGS-only file WITH an .srt beside it was not listed — this is the Plex-transcode case")
	}
	if !needsOf(mi, dp, "hevc", false).RemuxOnly() {
		t.Error("dropping a subtitle track should be a copy, not a re-encode")
	}
}

// Sidecar discovery follows the Subtitles module's naming: "<name>.<lang>.srt", with an
// optional ".forced" between.
func TestSidecarLangs(t *testing.T) {
	dir := t.TempDir()
	video := filepath.Join(dir, "Ted Lasso - S02E01 - Goodbye Earl - 2160p WEB-DL.mkv")
	for _, name := range []string{
		"Ted Lasso - S02E01 - Goodbye Earl - 2160p WEB-DL.en.srt",
		"Ted Lasso - S02E01 - Goodbye Earl - 2160p WEB-DL.fr.forced.srt",
		"Ted Lasso - S02E02 - Lavender - 2160p WEB-DL.en.srt", // a different episode
		"Ted Lasso - S02E01 - Goodbye Earl - 2160p WEB-DL.nfo",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := sidecarLangs(video, nil)
	if strings.Join(got, ",") != "en,fr" && strings.Join(got, ",") != "fr,en" {
		t.Errorf("sidecarLangs = %v, want en and fr for this episode only", got)
	}
	// The cache serves a second episode in the same folder without another ReadDir —
	// and still answers correctly for it.
	cache := map[string][]string{}
	_ = sidecarLangs(video, cache)
	if _, ok := cache[dir]; !ok {
		t.Error("directory listing was not cached")
	}
	other := filepath.Join(dir, "Ted Lasso - S02E02 - Lavender - 2160p WEB-DL.mkv")
	if got := sidecarLangs(other, cache); len(got) != 1 || got[0] != "en" {
		t.Errorf("cached lookup for the other episode = %v, want [en]", got)
	}
}

package convert

import (
	"testing"
	"time"
)

// The Overview's figures are served from cache until the index changes, the settings
// they depend on change, or they get old — each of those must miss.
func TestLibCacheEntryFreshness(t *testing.T) {
	e := libCacheEntry[int]{gen: 3, key: "hevc", at: time.Now(), v: 42, ok: true}
	if !e.fresh(3, "hevc") {
		t.Error("an entry just written for the same gen/key was not fresh")
	}
	if e.fresh(4, "hevc") {
		t.Error("stayed fresh after an index write (gen moved)")
	}
	if e.fresh(3, "av1") {
		t.Error("stayed fresh after the settings it depends on changed (key moved)")
	}
	old := e
	old.at = time.Now().Add(-libCacheTTL - time.Second)
	if old.fresh(3, "hevc") {
		t.Error("stayed fresh past the TTL — sidecars written by Subtitles would never show")
	}
	var empty libCacheEntry[int]
	if empty.fresh(0, "") {
		t.Error("an entry that was never written counted as fresh")
	}
}

// Every index write bumps the generation, which is what makes a cached Overview notice an
// import, a convert, or a rescan.
func TestIndexWritesBumpGeneration(t *testing.T) {
	var ix libraryIndex
	if ix.gen.Load() != 0 {
		t.Fatal("fresh index has a non-zero generation")
	}
	ix.gen.Add(1)
	s := &Service{index: &ix}
	before := s.indexGen()
	s.invalidateLibraryCache()
	if s.indexGen() != before+1 {
		t.Errorf("invalidateLibraryCache: gen %d → %d, want +1", before, s.indexGen())
	}
	none := &Service{}
	if none.indexGen() != 0 {
		t.Error("a service without an index should report gen 0")
	}
	none.invalidateLibraryCache() // must not panic
}

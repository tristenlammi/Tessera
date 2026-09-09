package metadata

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tristenlammi/arrmada/internal/store"
)

func testDiskCache(t *testing.T) *DiskCache {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return NewDiskCache(st.DB())
}

// Fresh entries are served without fetching; past their TTL they're still served, and
// a refresh runs behind the response; errors are never stored.
func TestDiskCacheStaleWhileRevalidate(t *testing.T) {
	c := testDiskCache(t)
	ctx := context.Background()
	var fetches int32
	fetch := func(context.Context) ([]string, error) {
		n := atomic.AddInt32(&fetches, 1)
		return []string{"v" + string(rune('0'+n))}, nil
	}
	v, err := swr(ctx, c, "k", time.Hour, fetch)
	if err != nil || v[0] != "v1" {
		t.Fatalf("first: %v %v", v, err)
	}
	if v, _ := swr(ctx, c, "k", time.Hour, fetch); v[0] != "v1" || atomic.LoadInt32(&fetches) != 1 {
		t.Errorf("fresh entry was refetched: %v fetches=%d", v, fetches)
	}
	// Expire it in place (TTL 0), then read: stale value now, refresh in the background.
	c.put("k", []byte(`["stale"]`), 0)
	v, _ = swr(ctx, c, "k", time.Hour, fetch)
	if v[0] != "stale" {
		t.Errorf("stale entry not served while refreshing: %v", v)
	}
	deadline := time.Now().Add(2 * time.Second)
	for atomic.LoadInt32(&fetches) < 2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	time.Sleep(50 * time.Millisecond) // let the put land
	if v, _ := swr(ctx, c, "k", time.Hour, fetch); v[0] != "v2" {
		t.Errorf("background refresh didn't replace the stale value: %v", v)
	}
	// Errors are not cached.
	_, err = swr(ctx, c, "bad", time.Hour, func(context.Context) ([]string, error) { return nil, errors.New("boom") })
	if err == nil {
		t.Fatal("error swallowed")
	}
	if _, _, ok := c.get("bad"); ok {
		t.Error("an error was stored")
	}
	// Entries older than diskStaleMax are gone.
	_, _ = c.db.Exec(`UPDATE metadata_cache SET stored_at = ? WHERE key = 'k'`, time.Now().Add(-diskStaleMax-time.Hour).Unix())
	if _, _, ok := c.get("k"); ok {
		t.Error("a week-old entry was still served")
	}
	if _, err := swr(ctx, (*DiskCache)(nil), "x", time.Hour, fetch); err != nil {
		t.Errorf("nil cache should just fetch: %v", err)
	}
}

// The extra rows are plain TMDB lists; check the providers row picks the region's
// services in their order and the streaming query carries the right filters.
func TestTMDBProvidersAndNewOnProvider(t *testing.T) {
	var gotPaths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path+"?"+r.URL.RawQuery)
		switch {
		case strings.HasPrefix(r.URL.Path, "/watch/providers/movie"):
			_, _ = w.Write([]byte(`{"results":[
				{"provider_id":8,"provider_name":"Netflix","logo_path":"/n.png","display_priorities":{"AU":1,"US":0}},
				{"provider_id":99,"provider_name":"Elsewhere Only","logo_path":"/e.png","display_priorities":{"US":3}},
				{"provider_id":21,"provider_name":"Stan","logo_path":"/s.png","display_priorities":{"AU":0}}]}`))
		case strings.HasPrefix(r.URL.Path, "/discover/movie"):
			_, _ = w.Write([]byte(`{"results":[{"id":1,"title":"New Film","release_date":"2026-05-01","poster_path":"/p.jpg","vote_average":7.1,"vote_count":120,"genre_ids":[28]}]}`))
		case strings.HasPrefix(r.URL.Path, "/genre/"):
			_, _ = w.Write([]byte(`{"genres":[{"id":28,"name":"Action"}]}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	tm := NewTMDB("k")
	tm.base = srv.URL
	tm.SetRegionFunc(func() string { return "AU" })

	ps, err := tm.WatchProviders(context.Background(), "movie")
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Name != "Stan" || ps[1].Name != "Netflix" || ps[0].LogoURL != tmdbLogoBase+"/s.png" {
		t.Errorf("providers = %+v (want Stan then Netflix, AU only)", ps)
	}
	tm.WarmGenres(context.Background())
	items, err := tm.NewOnProvider(context.Background(), "movie", 8)
	if err != nil || len(items) != 1 || items[0].Title != "New Film" {
		t.Fatalf("new on provider: %v %+v", err, items)
	}
	if len(items[0].Genres) != 1 || items[0].Genres[0] != "Action" {
		t.Errorf("genres not attached to the card: %+v", items[0].Genres)
	}
	joined := strings.Join(gotPaths, "\n")
	for _, want := range []string{"watch_region=AU", "with_watch_providers=8", "with_watch_monetization_types=flatrate", "primary_release_date.gte="} {
		if !strings.Contains(joined, want) {
			t.Errorf("query missing %s in:\n%s", want, joined)
		}
	}
}

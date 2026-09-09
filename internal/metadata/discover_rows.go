package metadata

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strconv"
	"time"
)

// More Discover rows on top of trending/popular/upcoming: what's in cinemas, the
// best-rated, anime, hidden gems, titles from the user's own region, and what's new
// on each streaming service the region has. All are TMDB browse lists, cached the
// same way as the rest.

// DiscoverRows is the optional capability the extra rows come from.
type DiscoverRows interface {
	NowPlaying(ctx context.Context) ([]DiscoverItem, error)
	TopRated(ctx context.Context, media string) ([]DiscoverItem, error)
	Anime(ctx context.Context) ([]DiscoverItem, error)
	HiddenGems(ctx context.Context, media string) ([]DiscoverItem, error)
	FromRegion(ctx context.Context, media string) ([]DiscoverItem, error)
	WatchProviders(ctx context.Context, media string) ([]WatchProvider, error)
	NewOnProvider(ctx context.Context, media string, providerID int) ([]DiscoverItem, error)
}

// WatchProvider is a streaming service available in the region.
type WatchProvider struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	LogoURL string `json:"logo_url,omitempty"`
}

const (
	tmdbLogoBase     = "https://image.tmdb.org/t/p/w92"
	providerRowMax   = 8
	providerRowsTTL  = 24 * time.Hour
	hiddenGemMinVote = 7.3
)

// regionOrDefault is the discovery region, or TMDB's most-populated one when unset.
func (t *TMDB) regionOrDefault() string {
	if r := t.regionCode(); r != "" {
		return r
	}
	return "US"
}

// NowPlaying is what's in cinemas in the region this week.
func (t *TMDB) NowPlaying(ctx context.Context) ([]DiscoverItem, error) {
	q := url.Values{}
	q.Set("region", t.regionOrDefault())
	return t.cachedDiscoverListN(ctx, "/movie/now_playing", q, "movie", discoverListTTL, 0, true, 2)
}

// TopRated is TMDB's best-rated list, two pages so dedupe against the other rows
// still leaves a full strip.
func (t *TMDB) TopRated(ctx context.Context, media string) ([]DiscoverItem, error) {
	if tvish(media) {
		return t.cachedDiscoverListN(ctx, "/tv/top_rated", url.Values{}, "tv", discoverListTTL, browseMinVotes, true, 2)
	}
	q := url.Values{}
	q.Set("region", t.regionOrDefault())
	return t.cachedDiscoverListN(ctx, "/movie/top_rated", q, "movie", discoverListTTL, browseMinVotes, true, 2)
}

// Anime is popular Japanese animation on TV.
func (t *TMDB) Anime(ctx context.Context) ([]DiscoverItem, error) {
	q := url.Values{}
	q.Set("with_genres", "16")
	q.Set("with_origin_country", "JP")
	q.Set("sort_by", "popularity.desc")
	q.Set("vote_count.gte", strconv.Itoa(browseMinVotes))
	q.Set("include_adult", "false")
	return t.cachedDiscoverListN(ctx, "/discover/tv", q, "tv", discoverListTTL, browseMinVotes, true, 2)
}

// HiddenGems are well-rated titles that never got huge: a high average from a modest
// number of votes, recent enough to be findable.
func (t *TMDB) HiddenGems(ctx context.Context, media string) ([]DiscoverItem, error) {
	q := url.Values{}
	q.Set("sort_by", "vote_average.desc")
	q.Set("vote_average.gte", strconv.FormatFloat(hiddenGemMinVote, 'f', 1, 64))
	q.Set("vote_count.gte", "300")
	q.Set("vote_count.lte", "3000")
	q.Set("include_adult", "false")
	since := time.Now().AddDate(-12, 0, 0).Format("2006-01-02")
	if tvish(media) {
		q.Set("first_air_date.gte", since)
		q.Set("without_genres", noiseGenreCSV)
		return t.cachedDiscoverListN(ctx, "/discover/tv", q, "tv", discoverListTTL, 0, true, 2)
	}
	q.Set("primary_release_date.gte", since)
	q.Set("region", t.regionOrDefault())
	return t.cachedDiscoverListN(ctx, "/discover/movie", q, "movie", discoverListTTL, 0, true, 2)
}

// FromRegion is popular titles made in the discovery region. Nothing when no region
// is configured — a global "from your region" means nothing.
func (t *TMDB) FromRegion(ctx context.Context, media string) ([]DiscoverItem, error) {
	region := t.regionCode()
	if region == "" {
		return nil, nil
	}
	q := url.Values{}
	q.Set("with_origin_country", region)
	q.Set("sort_by", "popularity.desc")
	q.Set("vote_count.gte", "20")
	q.Set("include_adult", "false")
	if tvish(media) {
		q.Set("without_genres", noiseGenreCSV)
		return t.cachedDiscoverListN(ctx, "/discover/tv", q, "tv", discoverListTTL, 0, true, 2)
	}
	return t.cachedDiscoverListN(ctx, "/discover/movie", q, "movie", discoverListTTL, 0, true, 2)
}

// WatchProviders lists the region's streaming services, most prominent first.
func (t *TMDB) WatchProviders(ctx context.Context, media string) ([]WatchProvider, error) {
	region := t.regionOrDefault()
	seg := "movie"
	if tvish(media) {
		seg = "tv"
	}
	key := "providers:" + seg + ":" + region
	return swr(ctx, t.disk, key, providerRowsTTL, func(ctx context.Context) ([]WatchProvider, error) {
		q := url.Values{}
		q.Set("watch_region", region)
		body, err := t.get(ctx, "/watch/providers/"+seg, q)
		if err != nil {
			return nil, err
		}
		var payload struct {
			Results []struct {
				ID         int            `json:"provider_id"`
				Name       string         `json:"provider_name"`
				Logo       string         `json:"logo_path"`
				Priorities map[string]int `json:"display_priorities"`
				Priority   int            `json:"display_priority"`
			} `json:"results"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, err
		}
		type ranked struct {
			p    WatchProvider
			rank int
		}
		var list []ranked
		for _, r := range payload.Results {
			rank, ok := r.Priorities[region]
			if !ok {
				continue // not offered in this region
			}
			p := WatchProvider{ID: r.ID, Name: r.Name}
			if r.Logo != "" {
				p.LogoURL = tmdbLogoBase + r.Logo
			}
			list = append(list, ranked{p, rank})
		}
		sort.SliceStable(list, func(i, j int) bool { return list[i].rank < list[j].rank })
		out := make([]WatchProvider, 0, providerRowMax)
		for _, r := range list {
			if len(out) >= providerRowMax {
				break
			}
			out = append(out, r.p)
		}
		return out, nil
	})
}

// NewOnProvider is what a streaming service added to its catalogue in the region
// over the last year, most popular first.
func (t *TMDB) NewOnProvider(ctx context.Context, media string, providerID int) ([]DiscoverItem, error) {
	q := url.Values{}
	q.Set("watch_region", t.regionOrDefault())
	q.Set("with_watch_providers", strconv.Itoa(providerID))
	q.Set("with_watch_monetization_types", "flatrate")
	q.Set("sort_by", "popularity.desc")
	q.Set("vote_count.gte", "20")
	q.Set("include_adult", "false")
	since := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	if tvish(media) {
		q.Set("first_air_date.gte", since)
		q.Set("without_genres", noiseGenreCSV)
		return t.cachedDiscoverListN(ctx, "/discover/tv", q, "tv", discoverListTTL, 0, true, 2)
	}
	q.Set("primary_release_date.gte", since)
	return t.cachedDiscoverListN(ctx, "/discover/movie", q, "movie", discoverListTTL, 0, true, 2)
}

// genreNames maps TMDB genre ids to names, both media types together (ids don't
// overlap between them). It never fetches: the map is filled by WarmGenres (the
// Discover warm-up at startup) from TMDB, or here from the disk cache, so listing a
// row costs exactly the row's requests. Until it's loaded, cards carry no genres.
func (t *TMDB) genreNames(ctx context.Context) map[int]string {
	t.genreMu.Lock()
	defer t.genreMu.Unlock()
	if t.genreMap != nil {
		return t.genreMap
	}
	if t.disk != nil {
		if raw, _, ok := t.disk.get("genres:all"); ok {
			var m map[int]string
			if json.Unmarshal(raw, &m) == nil && len(m) > 0 {
				t.genreMap, t.genreAt = m, time.Now()
			}
		}
	}
	return t.genreMap
}

// WarmGenres loads (or refreshes, daily) the genre map from TMDB.
func (t *TMDB) WarmGenres(ctx context.Context) {
	t.genreMu.Lock()
	fresh := t.genreMap != nil && time.Since(t.genreAt) < 24*time.Hour
	t.genreMu.Unlock()
	if fresh {
		return
	}
	m, err := swr(ctx, t.disk, "genres:all", 24*time.Hour, func(ctx context.Context) (map[int]string, error) {
		out := map[int]string{}
		for _, media := range []string{"movie", "tv"} {
			gs, err := t.Genres(ctx, media)
			if err != nil {
				return nil, err
			}
			for _, g := range gs {
				out[g.ID] = g.Name
			}
		}
		return out, nil
	})
	if err != nil || len(m) == 0 {
		return
	}
	t.genreMu.Lock()
	t.genreMap, t.genreAt = m, time.Now()
	t.genreMu.Unlock()
}

func genreLabels(ids []int, names map[int]string) []string {
	if len(names) == 0 {
		return nil
	}
	var out []string
	for _, id := range ids {
		if n, ok := names[id]; ok && n != "" {
			out = append(out, n)
			if len(out) == 3 {
				break
			}
		}
	}
	return out
}

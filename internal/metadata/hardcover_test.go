package metadata

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// hcServer fakes Hardcover's GraphQL endpoint: it answers each query by the first
// matching substring in routes, records the Authorization header, and rejects an
// empty one like the real thing.
func hcServer(t *testing.T, routes map[string]string) (*Hardcover, *[]string) {
	t.Helper()
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auths = append(auths, r.Header.Get("Authorization"))
		body, _ := io.ReadAll(r.Body)
		var req struct {
			Query string `json:"query"`
		}
		_ = json.Unmarshal(body, &req)
		for needle, resp := range routes {
			if strings.Contains(req.Query, needle) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, resp)
				return
			}
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"errors":[{"message":"no route in test for query"}]}`)
	}))
	t.Cleanup(srv.Close)
	h := NewHardcoverFunc(func() string { return "testtoken" }, nil)
	h.endpoint = srv.URL
	return h, &auths
}

// The search index answers with a Typesense document: string ids, an image object,
// author names as a list. One hit per book — editions are already folded in.
func TestHardcoverSearchDecodesHits(t *testing.T) {
	results := `{"found":2,"hits":[
		{"document":{"id":"1245","title":"Dune","author_names":["Frank Herbert"],"release_year":1965,"image":{"url":"https://img/dune.jpg"}}},
		{"document":{"id":"99","title":"Dune / Dune Messiah","author_names":["Frank Herbert"],"release_year":1969,"image":"https://img/omnibus.jpg"}}
	]}`
	payload, _ := json.Marshal(map[string]any{"data": map[string]any{"search": map[string]any{"results": json.RawMessage(results)}}})
	h, auths := hcServer(t, map[string]string{"search(": string(payload)})

	got, err := h.SearchBooks(context.Background(), "dune")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "hc:1245" || got[0].Author != "Frank Herbert" || got[0].Year != 1965 || got[0].CoverURL != "https://img/dune.jpg" {
		t.Errorf("results = %+v (the omnibus should be filtered as a bundle)", got)
	}
	if len(*auths) == 0 || (*auths)[0] != "Bearer testtoken" {
		t.Errorf("auth header = %q, want the Bearer scheme added", *auths)
	}
	// results may also arrive as a JSON string containing the object.
	quoted, _ := json.Marshal(results)
	payload2, _ := json.Marshal(map[string]any{"data": map[string]any{"search": map[string]any{"results": json.RawMessage(quoted)}}})
	h2, _ := hcServer(t, map[string]string{"search(": string(payload2)})
	if got, err := h2.SearchBooks(context.Background(), "dune"); err != nil || len(got) != 1 {
		t.Errorf("string-wrapped results: %v %+v", err, got)
	}
}

// A token pasted with its "Bearer " prefix (how Hardcover's settings page hands it
// out) must not be doubled.
func TestHardcoverBearerPrefixNotDoubled(t *testing.T) {
	h, auths := hcServer(t, map[string]string{"search(": `{"data":{"search":{"results":{"hits":[]}}}}`})
	h.key = func() string { return "Bearer abc" }
	_, _ = h.SearchBooks(context.Background(), "x")
	if (*auths)[0] != "Bearer abc" {
		t.Errorf("auth = %q", (*auths)[0])
	}
}

// A book Hardcover has merged into another comes back as the canonical one, under
// the canonical key — the library can't end up with both.
func TestHardcoverGetBookFollowsCanonical(t *testing.T) {
	dupe := `{"data":{"books":[{"id":50,"title":"Dune (old duplicate)","canonical_id":1245,"contributions":[]}]}}`
	canon := `{"data":{"books":[{"id":1245,"title":"Dune","release_year":1965,"description":"Arrakis.","image":{"url":"https://img/dune.jpg"},
		"contributions":[{"author":{"id":7,"name":"Frank Herbert"}}],
		"cached_tags":{"Genre":[{"tag":"Science Fiction"},{"tag":"Classics"}],"Mood":[{"tag":"Epic"}]},
		"book_series":[{"position":1,"series":{"name":"Dune"}}]}]}}`
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls++
		if strings.Contains(string(body), `"id":50`) {
			_, _ = io.WriteString(w, dupe)
			return
		}
		_, _ = io.WriteString(w, canon)
	}))
	defer srv.Close()
	h := NewHardcoverFunc(func() string { return "k" }, nil)
	h.endpoint = srv.URL

	d, err := h.GetBook(context.Background(), "hc:50")
	if err != nil {
		t.Fatal(err)
	}
	if d.Key != "hc:1245" || d.Title != "Dune" || d.Author != "Frank Herbert" || d.Year != 1965 {
		t.Errorf("details = %+v", d.BookResult)
	}
	if d.SeriesName != "Dune" || d.SeriesPosition != 1 {
		t.Errorf("series = %q #%v", d.SeriesName, d.SeriesPosition)
	}
	if len(d.Subjects) != 2 || d.Subjects[0] != "Science Fiction" {
		t.Errorf("subjects = %v, want the Genre tags only", d.Subjects)
	}
	if calls != 2 {
		t.Errorf("%d requests, want 2 (duplicate, then canonical)", calls)
	}
}

func TestHardcoverRejectedKeyIsExplained(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer srv.Close()
	h := NewHardcoverFunc(func() string { return "expired" }, nil)
	h.endpoint = srv.URL
	if _, err := h.SearchBooks(context.Background(), "x"); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Errorf("err = %v, want a hint about the key", err)
	}
}

// stubBooks is a BookProvider that records what it was asked and answers canned results.
type stubBooks struct {
	name    string
	results []BookResult
	calls   []string
}

func (s *stubBooks) Available() bool { return true }
func (s *stubBooks) SearchBooks(ctx context.Context, q string) ([]BookResult, error) {
	s.calls = append(s.calls, "search:"+q)
	return s.results, nil
}
func (s *stubBooks) GetBook(ctx context.Context, key string) (*BookDetails, error) {
	s.calls = append(s.calls, "get:"+key)
	return &BookDetails{BookResult: BookResult{Key: key, Title: s.name}}, nil
}
func (s *stubBooks) Covers(ctx context.Context, key, title, author string) ([]string, error) {
	s.calls = append(s.calls, "covers:"+key)
	return nil, nil
}
func (s *stubBooks) SearchAuthors(ctx context.Context, q string) ([]AuthorResult, error) {
	s.calls = append(s.calls, "authors:"+q)
	return nil, nil
}
func (s *stubBooks) AuthorWorks(ctx context.Context, key string, limit int) ([]BookResult, error) {
	s.calls = append(s.calls, "works:"+key)
	return s.results, nil
}
func (s *stubBooks) TrendingBooks(ctx context.Context) ([]BookResult, error) {
	s.calls = append(s.calls, "trending")
	return s.results, nil
}
func (s *stubBooks) BooksBySubject(ctx context.Context, subject string, limit int) ([]BookResult, error) {
	s.calls = append(s.calls, "subject:"+subject)
	return s.results, nil
}

// Without a key everything goes to Open Library. With one, searches go to Hardcover,
// Open Library stays reachable on request, and a stored key routes to its own source.
func TestBookSourcesRouting(t *testing.T) {
	ol := &stubBooks{name: "ol", results: []BookResult{{Key: "OL1W", Title: "From OL"}}}
	empty := `{"data":{"search":{"results":{"hits":[]}}}}`
	hit := `{"data":{"search":{"results":{"hits":[{"document":{"id":"5","title":"From HC","author_names":["A"]}}]}}}}`
	hc, _ := hcServer(t, map[string]string{"search(": hit})
	key := ""
	hc.key = func() string { return key }
	src := NewBookSources(hc, ol)
	ctx := context.Background()

	if src.Source() != SourceOpenLibrary {
		t.Errorf("no key: source = %q", src.Source())
	}
	if r, _ := src.SearchBooks(ctx, "q"); len(r) != 1 || r[0].Key != "OL1W" {
		t.Errorf("no key: search went to %+v", r)
	}

	key = "tok"
	if src.Source() != SourceHardcover {
		t.Errorf("with key: source = %q", src.Source())
	}
	if r, err := src.SearchBooks(ctx, "q"); err != nil || len(r) != 1 || r[0].Key != "hc:5" {
		t.Errorf("with key: search = %+v %v", r, err)
	}
	if r, _ := src.SearchBooksFrom(ctx, SourceOpenLibrary, "q"); len(r) != 1 || r[0].Key != "OL1W" {
		t.Errorf("explicit Open Library search = %+v", r)
	}
	// Keys route to their issuer regardless of the current source.
	if _, err := src.GetBook(ctx, "OL1W"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(ol.calls, ","), "get:OL1W") {
		t.Errorf("an Open Library key was not sent to Open Library: %v", ol.calls)
	}
	if _, err := src.AuthorWorks(ctx, "OL23919A", 0); err != nil || !strings.Contains(strings.Join(ol.calls, ","), "works:OL23919A") {
		t.Errorf("Open Library author key routed elsewhere: %v %v", err, ol.calls)
	}

	// Hardcover empty → Open Library answers.
	hc2, _ := hcServer(t, map[string]string{"search(": empty})
	hc2.key = func() string { return "tok" }
	src2 := NewBookSources(hc2, ol)
	if r, _ := src2.SearchBooks(ctx, "obscure"); len(r) != 1 || r[0].Key != "OL1W" {
		t.Errorf("empty Hardcover result should fall back to Open Library, got %+v", r)
	}
}

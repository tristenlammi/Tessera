package metadata

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeAndFindISBN(t *testing.T) {
	for in, want := range map[string]string{
		"978-0-441-17271-9":    "9780441172719",
		"0441172717":           "0441172717",
		"0-8044-2957-X":        "080442957X",
		"9780441172710":        "", // bad checksum
		"1234567890":           "", // bad checksum
		"Dune [9780441172719]": "9780441172719",
		"dune_0441172717.epub": "0441172717",
		"Dune (1965) 1080p":    "",
		"Frank Herbert - Dune - 9780441172719 - retail": "9780441172719",
	} {
		if got := FindISBN(in); got != want {
			t.Errorf("FindISBN(%q) = %q, want %q", in, got, want)
		}
	}
}

// A series comes back with one entry per position, in order, tagged with the series.
func TestHardcoverSeriesBooks(t *testing.T) {
	series := `{"data":{"series":[{"id":9,"name":"Dune","books_count":6,"book_series":[
		{"position":1,"book":{"id":1,"title":"Dune","release_year":1965,"contributions":[{"author":{"id":7,"name":"Frank Herbert"}}]}},
		{"position":2,"book":{"id":2,"title":"Dune Messiah","release_year":1969,"contributions":[{"author":{"id":7,"name":"Frank Herbert"}}]}},
		{"position":3,"book":{"id":3,"title":"Children of Dune","release_year":1976,"contributions":[]}}
	]}]}}`
	h, _ := hcServer(t, map[string]string{"series(": series})
	info, err := h.SeriesBooks(context.Background(), "hc:s:9")
	if err != nil {
		t.Fatal(err)
	}
	if info.Name != "Dune" || info.Count != 6 || len(info.Books) != 3 {
		t.Fatalf("info = %+v", info)
	}
	if info.Books[1].Key != "hc:2" || info.Books[1].SeriesPosition != 2 || info.Books[1].SeriesName != "Dune" {
		t.Errorf("second entry = %+v", info.Books[1])
	}
	if _, err := h.SeriesBooks(context.Background(), "OL1W"); err != ErrNotSupported {
		t.Errorf("non-Hardcover series key: err = %v", err)
	}
}

// If the compilation filter on the join row is refused, the query is retried without it.
func TestHardcoverSeriesRetriesWithoutCompilationFilter(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		calls++
		if strings.Contains(string(body), "compilation") {
			_, _ = io.WriteString(w, `{"errors":[{"message":"field 'compilation' not found in type: 'book_series_bool_exp'"}]}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"series":[{"id":9,"name":"Dune","books_count":1,"book_series":[{"position":1,"book":{"id":1,"title":"Dune","contributions":[]}}]}]}}`)
	}))
	defer srv.Close()
	h := NewHardcoverFunc(func() string { return "k" }, nil)
	h.endpoint = srv.URL
	info, err := h.SeriesBooks(context.Background(), "hc:s:9")
	if err != nil || len(info.Books) != 1 {
		t.Fatalf("err=%v info=%+v", err, info)
	}
	if calls != 2 {
		t.Errorf("%d calls, want 2 (with the filter, then without)", calls)
	}
}

// Similar books: the id list is fetched, then the books, and the catalogue's order kept.
func TestHardcoverSimilarBooksKeepsOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if strings.Contains(string(body), "cached_similar_book_ids") {
			_, _ = io.WriteString(w, `{"data":{"books":[{"cached_similar_book_ids":[30,10,20]}]}}`)
			return
		}
		_, _ = io.WriteString(w, `{"data":{"books":[
			{"id":10,"title":"Ten","contributions":[]},{"id":20,"title":"Twenty","contributions":[]},{"id":30,"title":"Thirty","contributions":[]}]}}`)
	}))
	defer srv.Close()
	h := NewHardcoverFunc(func() string { return "k" }, nil)
	h.endpoint = srv.URL
	got, err := h.SimilarBooks(context.Background(), "hc:1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[0].Key != "hc:30" || got[1].Key != "hc:10" || got[2].Key != "hc:20" {
		t.Errorf("similar = %+v", got)
	}
}

func TestHardcoverAuthorDetailAndISBN(t *testing.T) {
	h, _ := hcServer(t, map[string]string{
		"authors(":  `{"data":{"authors":[{"id":7,"name":"Frank Herbert","bio":"Wrote Dune.","books_count":40,"born_year":1920,"death_year":1986,"image":{"url":"https://img/fh.jpg"}}]}}`,
		"editions(": `{"data":{"editions":[{"book":{"id":1,"title":"Dune","release_year":1965,"contributions":[{"author":{"id":7,"name":"Frank Herbert"}}]}}]}}`,
	})
	a, err := h.AuthorDetail(context.Background(), "hc:a:7")
	if err != nil {
		t.Fatal(err)
	}
	if a.Name != "Frank Herbert" || a.ImageURL != "https://img/fh.jpg" || a.Bio != "Wrote Dune." || a.BirthDate != "1920–1986" || a.WorkCount != 40 {
		t.Errorf("author = %+v", a)
	}
	r, err := h.BookByISBN(context.Background(), "978-0-441-17271-9")
	if err != nil || r == nil || r.Key != "hc:1" {
		t.Errorf("isbn: %v %+v", err, r)
	}
	if _, err := h.BookByISBN(context.Background(), "not-an-isbn"); err == nil {
		t.Error("an invalid ISBN was looked up")
	}
}

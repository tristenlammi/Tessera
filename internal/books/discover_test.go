package books

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/tristenlammi/arrmada/internal/metadata"
)

// stubCatalogue is a BookProvider with the optional catalogue capabilities, answering
// canned similar-books lists per key.
type stubCatalogue struct {
	similar map[string][]metadata.BookResult
	calls   []string
}

func (s *stubCatalogue) Available() bool { return true }
func (s *stubCatalogue) SearchBooks(context.Context, string) ([]metadata.BookResult, error) {
	return nil, nil
}
func (s *stubCatalogue) GetBook(_ context.Context, key string) (*metadata.BookDetails, error) {
	return &metadata.BookDetails{BookResult: metadata.BookResult{Key: key}}, nil
}
func (s *stubCatalogue) Covers(context.Context, string, string, string) ([]string, error) {
	return nil, nil
}
func (s *stubCatalogue) SearchAuthors(context.Context, string) ([]metadata.AuthorResult, error) {
	return nil, nil
}
func (s *stubCatalogue) AuthorWorks(context.Context, string, int) ([]metadata.BookResult, error) {
	return nil, nil
}
func (s *stubCatalogue) TrendingBooks(context.Context) ([]metadata.BookResult, error) {
	return nil, nil
}
func (s *stubCatalogue) BooksBySubject(context.Context, string, int) ([]metadata.BookResult, error) {
	return nil, nil
}
func (s *stubCatalogue) SeriesBooks(context.Context, string) (*metadata.SeriesInfo, error) {
	return nil, metadata.ErrNotSupported
}
func (s *stubCatalogue) AuthorDetail(context.Context, string) (*metadata.AuthorResult, error) {
	return nil, metadata.ErrNotSupported
}
func (s *stubCatalogue) SimilarBooks(_ context.Context, key string) ([]metadata.BookResult, error) {
	s.calls = append(s.calls, key)
	return s.similar[key], nil
}
func (s *stubCatalogue) BookByISBN(context.Context, string) (*metadata.BookResult, error) {
	return nil, nil
}

// "Because you own X" rows come from Hardcover-keyed books, leave out what the library
// already holds (by key or by title+author), and skip seeds with too little to show.
func TestRecommendedRowsExcludeOwnedBooks(t *testing.T) {
	repo, ctx := historyRepo(t)
	cat := &stubCatalogue{similar: map[string][]metadata.BookResult{
		"hc:1": {
			{Key: "hc:2", Title: "Dune Messiah", Author: "Frank Herbert"}, // owned by key
			{Key: "hc:9", Title: "The Hobbit", Author: "J.R.R. Tolkien"},  // owned under an OL key
			{Key: "hc:3", Title: "Hyperion", Author: "Dan Simmons"},
			{Key: "hc:4", Title: "Foundation", Author: "Isaac Asimov"},
			{Key: "hc:5", Title: "Neuromancer", Author: "William Gibson"},
		},
		"hc:2": {{Key: "hc:6", Title: "Only one", Author: "X"}}, // too thin for a row
	}}
	s := &Service{repo: repo, meta: cat, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	dune, _ := repo.Create(ctx, Book{OLKey: "hc:1", Title: "Dune", Author: "Frank Herbert"})
	_ = repo.SetEdition(ctx, dune.ID, KindEbook, "/b/dune.epub", "EPUB", 1, 1)
	_, _ = repo.Create(ctx, Book{OLKey: "hc:2", Title: "Dune Messiah", Author: "Frank Herbert"})
	_, _ = repo.Create(ctx, Book{OLKey: "OL7W", Title: "The Hobbit", Author: "J. R. R. Tolkien"})

	rows, err := s.recommendedAt(ctx, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)) // day 1 → offset 1 % 3 = 1
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1 (Dune's; Messiah's is too thin, the Hobbit isn't on Hardcover): %+v", len(rows), rows)
	}
	r := rows[0]
	if r.Seed != "Dune" || r.Title != "Because you own Dune" {
		t.Errorf("row = %+v", r)
	}
	if len(r.Books) != 3 {
		t.Errorf("got %d books, want 3 (owned ones removed): %+v", len(r.Books), r.Books)
	}
	for _, b := range r.Books {
		if b.Key == "hc:2" || b.Key == "hc:9" {
			t.Errorf("owned book %q was recommended", b.Title)
		}
	}
	// Open Library-keyed books are never seeds: only hc:1 and hc:2 were asked.
	for _, c := range cat.calls {
		if c == "OL7W" {
			t.Error("an Open Library key was used as a seed")
		}
	}
}

func TestRecommendedWithoutCatalogueSupport(t *testing.T) {
	repo, ctx := historyRepo(t)
	s := &Service{repo: repo, meta: &plainProvider{}, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if _, err := s.Recommended(ctx); err != metadata.ErrNotSupported {
		t.Errorf("err = %v, want ErrNotSupported", err)
	}
}

type plainProvider struct{ stubCatalogue }

// plainProvider hides the optional capabilities by not being a catalogueExtras — it
// embeds the stub but Go's interface satisfaction is structural, so shadow one method
// with a different signature to break it.
func (p *plainProvider) SimilarBooks(context.Context) {}

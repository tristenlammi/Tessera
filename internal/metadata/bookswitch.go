package metadata

import (
	"context"
	"fmt"
	"strings"
)

// BookSources is the Books module's provider: Open Library out of the box, Hardcover
// the moment a key is configured, and Open Library still reachable on demand ("show
// Open Library results too") for the odd title Hardcover hasn't got. Stored keys route
// to whichever catalogue issued them, so a library built on Open Library keeps working
// after the key goes in, and books added from Hardcover keep working if it's removed
// (they just stop refreshing).
type BookSources struct {
	hardcover *Hardcover
	openlib   BookProvider // Open Library (with its Google Books fallback)
}

const (
	SourceHardcover   = "hardcover"
	SourceOpenLibrary = "openlibrary"
)

func NewBookSources(hardcover *Hardcover, openlib BookProvider) *BookSources {
	return &BookSources{hardcover: hardcover, openlib: openlib}
}

// Source names the catalogue searches go to right now.
func (s *BookSources) Source() string {
	if s.hardcover != nil && s.hardcover.Available() {
		return SourceHardcover
	}
	return SourceOpenLibrary
}

func (s *BookSources) primary() BookProvider {
	if s.Source() == SourceHardcover {
		return s.hardcover
	}
	return s.openlib
}

// byKey picks the provider that issued a key.
func (s *BookSources) byKey(key string) BookProvider {
	if IsHardcoverKey(key) && s.hardcover != nil {
		return s.hardcover
	}
	return s.openlib
}

func (s *BookSources) Available() bool { return true }

// VerifyHardcover exercises the Hardcover key end to end (settings "Test" button).
func (s *BookSources) VerifyHardcover(ctx context.Context) (string, error) {
	if s.hardcover == nil {
		return "", fmt.Errorf("Hardcover is not configured")
	}
	return s.hardcover.Verify(ctx)
}

// SearchBooks asks the current source; when that's Hardcover and it errors or comes
// up empty, Open Library answers instead.
func (s *BookSources) SearchBooks(ctx context.Context, query string) ([]BookResult, error) {
	if s.Source() == SourceHardcover {
		r, err := s.hardcover.SearchBooks(ctx, query)
		if err == nil && len(r) > 0 {
			return r, nil
		}
	}
	return s.openlib.SearchBooks(ctx, query)
}

// SearchBooksFrom searches a named source regardless of what's primary — the "also
// show Open Library results" button. An unknown or empty source means the usual path.
func (s *BookSources) SearchBooksFrom(ctx context.Context, source, query string) ([]BookResult, error) {
	switch strings.ToLower(source) {
	case SourceOpenLibrary:
		return s.openlib.SearchBooks(ctx, query)
	case SourceHardcover:
		if s.hardcover != nil && s.hardcover.Available() {
			return s.hardcover.SearchBooks(ctx, query)
		}
		return nil, nil
	}
	return s.SearchBooks(ctx, query)
}

func (s *BookSources) GetBook(ctx context.Context, key string) (*BookDetails, error) {
	return s.byKey(key).GetBook(ctx, key)
}

func (s *BookSources) Covers(ctx context.Context, key, title, author string) ([]string, error) {
	return s.byKey(key).Covers(ctx, key, title, author)
}

func (s *BookSources) SearchAuthors(ctx context.Context, query string) ([]AuthorResult, error) {
	if s.Source() == SourceHardcover {
		r, err := s.hardcover.SearchAuthors(ctx, query)
		if err == nil && len(r) > 0 {
			return r, nil
		}
	}
	return s.openlib.SearchAuthors(ctx, query)
}

func (s *BookSources) AuthorWorks(ctx context.Context, authorKey string, limit int) ([]BookResult, error) {
	if strings.HasPrefix(authorKey, hcAuthorPrefix) && s.hardcover != nil {
		return s.hardcover.AuthorWorks(ctx, authorKey, limit)
	}
	return s.openlib.AuthorWorks(ctx, authorKey, limit)
}

func (s *BookSources) TrendingBooks(ctx context.Context) ([]BookResult, error) {
	if s.Source() == SourceHardcover {
		r, err := s.hardcover.TrendingBooks(ctx)
		if err == nil && len(r) > 0 {
			return r, nil
		}
	}
	return s.openlib.TrendingBooks(ctx)
}

func (s *BookSources) BooksBySubject(ctx context.Context, subject string, limit int) ([]BookResult, error) {
	if s.Source() == SourceHardcover {
		r, err := s.hardcover.BooksBySubject(ctx, subject, limit)
		if err == nil && len(r) > 0 {
			return r, nil
		}
	}
	return s.openlib.BooksBySubject(ctx, subject, limit)
}

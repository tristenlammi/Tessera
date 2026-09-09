package books

import (
	"testing"

	"github.com/tristenlammi/arrmada/internal/metadata"
)

// The re-match must land on the catalogue's entry for the same book across the ways
// catalogues render it, and must not land on a different book of the same title.
func TestMatchUpgrade(t *testing.T) {
	book := Book{Title: "Dust", Author: "Hugh Howey"}
	results := []metadata.BookResult{
		{Key: "hc:1", Title: "Dust", Author: "Elizabeth Bear"},
		{Key: "hc:2", Title: "Dust (Silo, #3)", Author: "Howey, Hugh"},
		{Key: "hc:3", Title: "Wool", Author: "Hugh Howey"},
	}
	if m := matchUpgrade(book, results); m == nil || m.Key != "hc:2" {
		t.Errorf("matched %+v, want hc:2 (same title, author shares 'howey')", m)
	}
	// Exact key match wins over an overlap.
	exact := append([]metadata.BookResult{{Key: "hc:9", Title: "Dust", Author: "Hugh Howey"}}, results...)
	if m := matchUpgrade(book, exact); m == nil || m.Key != "hc:9" {
		t.Errorf("matched %+v, want the exact hc:9", m)
	}
	// Same title, different author, no overlap: not a match.
	if m := matchUpgrade(book, results[:1]); m != nil {
		t.Errorf("matched a different author's Dust: %+v", m)
	}
	// No author on the library side: the single same-title result is taken.
	if m := matchUpgrade(Book{Title: "Dust"}, results[1:2]); m == nil || m.Key != "hc:2" {
		t.Errorf("authorless book didn't take the sole same-title result: %+v", m)
	}
	if m := matchUpgrade(Book{Title: "Dust"}, results[:2]); m == nil || m.Key != "hc:1" {
		t.Errorf("authorless book with two same-title results should take the first: %+v", m)
	}
}

func TestAuthorsOverlap(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want bool
	}{
		{"Hugh Howey", "Howey, Hugh", true},
		{"H. Howey", "Hugh Howey", true},
		{"J.K. Rowling", "Rowling, J. K.", true},
		{"Hugh Howey", "Elizabeth Bear", false},
		{"Jo Nesbø", "Jo Smith", false}, // "jo" is too short to count
	} {
		if got := authorsOverlap(c.a, c.b); got != c.want {
			t.Errorf("authorsOverlap(%q, %q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

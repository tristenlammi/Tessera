package automation

import (
	"path/filepath"

	"github.com/tristenlammi/arrmada/internal/library"
	"github.com/tristenlammi/arrmada/internal/metadata"
)

// scanISBN finds an ISBN in a scanned book folder — its name or any file's name —
// which many retail and calibre-exported folders carry ("Dune (9780441172719).epub").
// An ISBN identifies the book exactly, so the scan tries it before matching by title.
func scanISBN(bf library.BookFolder) string {
	if isbn := metadata.FindISBN(bf.Title); isbn != "" {
		return isbn
	}
	for _, files := range [][]library.FoundFile{bf.Ebooks, bf.Audiobooks} {
		for _, f := range files {
			if isbn := metadata.FindISBN(filepath.Base(f.Path)); isbn != "" {
				return isbn
			}
		}
	}
	return ""
}

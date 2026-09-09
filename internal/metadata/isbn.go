package metadata

import "regexp"

// isbnCandidate finds runs that look like an ISBN in a folder or file name — digits
// with optional hyphens/spaces, 10 or 13 long, an X allowed as the last ISBN-10 digit.
var isbnCandidate = regexp.MustCompile(`(?i)(?:97[89][-\s]?)?(?:\d[-\s]?){9}[\dX]`)

// FindISBN returns the first valid ISBN embedded in text (e.g. "Dune [9780441172719]"
// or "dune_0441172717.epub"), normalised, or "" when there is none.
func FindISBN(text string) string {
	for _, m := range isbnCandidate.FindAllString(text, -1) {
		if isbn := NormalizeISBN(m); isbn != "" {
			return isbn
		}
	}
	return ""
}

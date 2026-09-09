package books

import (
	"io"
	"log/slog"
	"testing"
)

// Three catalogue entries for one novel become one row: the one with files is kept,
// and monitoring, series and the missing edition come along from the others.
func TestMergeDuplicatesKeepsTheOneWithFiles(t *testing.T) {
	repo, ctx := historyRepo(t)
	s := &Service{repo: repo, log: slog.New(slog.NewTextHandler(io.Discard, nil))}

	first, _ := repo.Create(ctx, Book{OLKey: "OL1W", Title: "Dune", Author: "Frank Herbert", Monitored: true})
	second, _ := repo.Create(ctx, Book{OLKey: "OL2W", Title: "Dune (Dune Chronicles, #1)", Author: "Herbert, Frank", Description: "Arrakis."})
	third, _ := repo.Create(ctx, Book{OLKey: "gb:abc", Title: "Dune: Deluxe Edition", Author: "Frank Herbert"})
	other, _ := repo.Create(ctx, Book{OLKey: "OL9W", Title: "Dune Messiah", Author: "Frank Herbert"})
	if err := repo.SetEdition(ctx, second.ID, KindEbook, "/books/dune.epub", "EPUB", 1234, 1); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetEdition(ctx, third.ID, KindAudiobook, "/books/dune.m4b", "M4B", 99999, 1); err != nil {
		t.Fatal(err)
	}
	_ = repo.SetSeries(ctx, first.ID, "Dune", 1)

	n, err := s.MergeDuplicates(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("merged %d, want 2", n)
	}
	list, _ := repo.List(ctx)
	if len(list) != 2 {
		t.Fatalf("%d books left, want 2 (Dune + Dune Messiah): %+v", len(list), list)
	}
	var keeper Book
	for _, b := range list {
		if b.ID != other.ID {
			keeper = b
		}
	}
	// The ebook row won (it had a file); the audiobook, monitoring and series came across.
	if keeper.ID != second.ID {
		t.Errorf("kept id %d, want the row that had a file (%d)", keeper.ID, second.ID)
	}
	if keeper.Ebook == nil || keeper.Audiobook == nil {
		t.Errorf("editions not merged: ebook=%v audiobook=%v", keeper.Ebook != nil, keeper.Audiobook != nil)
	}
	if !keeper.Monitored {
		t.Error("monitoring not carried over")
	}
	if keeper.SeriesName != "Dune" || keeper.SeriesPosition != 1 {
		t.Errorf("series not carried over: %q #%v", keeper.SeriesName, keeper.SeriesPosition)
	}
	if keeper.Description != "Arrakis." {
		t.Errorf("description lost: %q", keeper.Description)
	}
	evs, _ := repo.Events(ctx, keeper.ID, 10)
	if len(evs) != 2 || evs[0].Event != "merged" {
		t.Errorf("merge not recorded on the keeper: %+v", evs)
	}
	// Running again finds nothing.
	if n, _ := s.MergeDuplicates(ctx); n != 0 {
		t.Errorf("second pass merged %d", n)
	}
}

// Adding a book that is already there under another catalogue's key is refused with
// the existing row, not added twice.
func TestFindDuplicateAcrossCatalogues(t *testing.T) {
	repo, ctx := historyRepo(t)
	s := &Service{repo: repo, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	existing, _ := repo.Create(ctx, Book{OLKey: "OL1W", Title: "The Hobbit", Author: "J.R.R. Tolkien"})
	if b, ok := s.findDuplicate(ctx, "Hobbit: 75th Anniversary Edition", "Tolkien, J. R. R."); !ok || b.ID != existing.ID {
		t.Errorf("duplicate not found: ok=%v id=%d", ok, b.ID)
	}
	if _, ok := s.findDuplicate(ctx, "The Hobbit", ""); ok {
		t.Error("a title with no author matched a book that has one")
	}
	if _, ok := s.findDuplicate(ctx, "The Silmarillion", "J.R.R. Tolkien"); ok {
		t.Error("a different title matched")
	}
}

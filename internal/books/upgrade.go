package books

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/tristenlammi/arrmada/internal/metadata"
)

// Upgrading the library to Hardcover.
//
// Books added before a Hardcover key went in carry Open Library (or Google Books)
// keys, so they still refresh from there — no series, the odd duplicate work, weaker
// covers. The upgrade re-matches each of them to Hardcover by title and author,
// rewrites the row to the Hardcover entry (key, description, cover, subjects, series)
// and, when two old rows land on the same Hardcover book, folds them into one.
// Files, monitoring and the quality profile are never touched.

// UpgradeStatus is the background job's progress, for the Books page.
type UpgradeStatus struct {
	Running   bool   `json:"running"`
	Total     int    `json:"total"`
	Done      int    `json:"done"`
	Upgraded  int    `json:"upgraded"`
	Merged    int    `json:"merged"`
	Unmatched int    `json:"unmatched"`
	StartedAt int64  `json:"started_at,omitempty"`
	EndedAt   int64  `json:"ended_at,omitempty"`
	Error     string `json:"error,omitempty"`
}

type upgradeState struct {
	mu     sync.Mutex
	status UpgradeStatus
}

// Upgradable counts the books not yet on Hardcover keys.
func (s *Service) Upgradable(ctx context.Context) int {
	list, err := s.repo.List(ctx)
	if err != nil {
		return 0
	}
	n := 0
	for _, b := range list {
		if !metadata.IsHardcoverKey(b.OLKey) {
			n++
		}
	}
	return n
}

// UpgradeStatus reports the job's progress.
func (s *Service) UpgradeStatus() UpgradeStatus {
	s.upgrade.mu.Lock()
	defer s.upgrade.mu.Unlock()
	return s.upgrade.status
}

// StartUpgrade begins re-matching in the background. False when one is already running
// or Hardcover isn't the current source.
func (s *Service) StartUpgrade(ctx context.Context) bool {
	if s.MetadataSource() != metadata.SourceHardcover {
		return false
	}
	s.upgrade.mu.Lock()
	if s.upgrade.status.Running {
		s.upgrade.mu.Unlock()
		return false
	}
	s.upgrade.status = UpgradeStatus{Running: true, StartedAt: time.Now().Unix()}
	s.upgrade.mu.Unlock()
	go s.runUpgrade(ctx)
	return true
}

func (s *Service) setUpgrade(fn func(*UpgradeStatus)) {
	s.upgrade.mu.Lock()
	fn(&s.upgrade.status)
	s.upgrade.mu.Unlock()
}

func (s *Service) runUpgrade(ctx context.Context) {
	defer s.setUpgrade(func(st *UpgradeStatus) { st.Running = false; st.EndedAt = time.Now().Unix() })
	list, err := s.repo.List(ctx)
	if err != nil {
		s.setUpgrade(func(st *UpgradeStatus) { st.Error = err.Error() })
		return
	}
	var todo []Book
	for _, b := range list {
		if !metadata.IsHardcoverKey(b.OLKey) {
			todo = append(todo, b)
		}
	}
	s.setUpgrade(func(st *UpgradeStatus) { st.Total = len(todo) })
	for _, b := range todo {
		if ctx.Err() != nil {
			s.setUpgrade(func(st *UpgradeStatus) { st.Error = "stopped" })
			return
		}
		outcome, err := s.upgradeOne(ctx, b)
		if errors.Is(err, metadata.ErrHardcoverBudget) {
			s.setUpgrade(func(st *UpgradeStatus) { st.Error = err.Error() })
			return
		}
		s.setUpgrade(func(st *UpgradeStatus) {
			st.Done++
			switch outcome {
			case "upgraded":
				st.Upgraded++
			case "merged":
				st.Merged++
			default:
				st.Unmatched++
			}
		})
	}
	s.log.Info("books: upgrade to Hardcover finished", "total", len(todo))
}

// upgradeOne re-matches a single book. Outcomes: "upgraded", "merged", "unmatched".
func (s *Service) upgradeOne(ctx context.Context, b Book) (string, error) {
	sl, ok := s.meta.(sourceLookup)
	if !ok {
		return "unmatched", nil
	}
	q := strings.TrimSpace(b.Title + " " + b.Author)
	results, err := sl.SearchBooksFrom(ctx, metadata.SourceHardcover, q)
	if err != nil {
		if errors.Is(err, metadata.ErrHardcoverBudget) {
			return "", err
		}
		s.log.Debug("books: upgrade search failed", "title", b.Title, "err", err)
		return "unmatched", nil
	}
	want := DedupeKey(b.Title, b.Author)
	var match *metadata.BookResult
	for i := range results {
		if DedupeKey(results[i].Title, results[i].Author) == want {
			match = &results[i]
			break
		}
	}
	if match == nil && b.Author == "" && len(results) > 0 && titleKey(results[0].Title) == titleKey(b.Title) {
		match = &results[0] // no author to check against: the top hit with the same title
	}
	if match == nil {
		return "unmatched", nil
	}
	d, err := s.meta.GetBook(ctx, match.Key)
	if err != nil {
		if errors.Is(err, metadata.ErrHardcoverBudget) {
			return "", err
		}
		return "unmatched", nil
	}
	// Another row already sits on this Hardcover book: this one is its duplicate.
	if other, ok := s.findByKey(ctx, d.Key); ok && other.ID != b.ID {
		keeper, dup := other, b
		if editionCount(b) > editionCount(other) {
			keeper, dup = b, other
			// Point the keeper at the Hardcover entry before folding the other in.
			if err := s.applyUpgrade(ctx, keeper, d); err != nil {
				return "unmatched", nil
			}
		}
		if err := s.foldInto(ctx, keeper, dup); err != nil {
			return "unmatched", nil
		}
		return "merged", nil
	}
	if err := s.applyUpgrade(ctx, b, d); err != nil {
		s.log.Warn("books: upgrade write failed", "title", b.Title, "err", err)
		return "unmatched", nil
	}
	return "upgraded", nil
}

// applyUpgrade rewrites a row to the Hardcover entry, keeping what the user set by
// hand: a custom cover (served from the app), and the year when Hardcover lacks one.
func (s *Service) applyUpgrade(ctx context.Context, b Book, d *metadata.BookDetails) error {
	nb := Book{
		OLKey: d.Key, Title: orStr(d.Title, b.Title), Author: orStr(d.Author, b.Author), Year: d.Year,
		CoverURL: orStr(d.CoverURL, b.CoverURL), Description: orStr(d.Description, b.Description), Subjects: d.Subjects,
	}
	if nb.Year == 0 {
		nb.Year = b.Year
	}
	if strings.HasPrefix(b.CoverURL, "/") { // an uploaded cover — the user chose it
		nb.CoverURL = b.CoverURL
	}
	if len(nb.Subjects) == 0 {
		nb.Subjects = b.Subjects
	}
	if err := s.repo.Rematch(ctx, b.ID, nb); err != nil {
		return err
	}
	if d.SeriesName != "" {
		_ = s.repo.SetSeriesRef(ctx, b.ID, d.SeriesName, d.SeriesPosition, d.SeriesKey)
	}
	s.repo.AddEvent(ctx, b.ID, "upgraded", "Re-matched to Hardcover ("+d.Key+")")
	return nil
}

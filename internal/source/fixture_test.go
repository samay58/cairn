package source

import (
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

func TestFixtureSourceCountAndByHandle(t *testing.T) {
	s := NewFixtureSource()
	if got := s.Count(); got != 25 {
		t.Errorf("Count() = %d, want 25", got)
	}
	c, err := s.ByHandle(2)
	if err != nil {
		t.Fatal(err)
	}
	if c.Kind != cards.KindQuote {
		t.Errorf("ByHandle(2).Kind = %q, want quote", c.Kind)
	}
	if _, err := s.ByHandle(99); err == nil {
		t.Error("ByHandle(99) should error")
	}
}

// Phase 0 demo queries: preserve exact fixture-driven results for golden
// stability.
func TestFixtureSourceSearchOAuthDemo(t *testing.T) {
	s := NewFixtureSource()
	m := s.Search("oauth", Filters{}, 0)
	if len(m) != 3 {
		t.Fatalf("oauth demo returned %d matches, want 3", len(m))
	}
	if m[0].WhyShown != "Matched on title and tag oauth." {
		t.Errorf("first why_shown = %q", m[0].WhyShown)
	}
}

func TestFixtureSourceSearchZZZEmpty(t *testing.T) {
	s := NewFixtureSource()
	if got := s.Search("zzz-empty", Filters{}, 0); got != nil {
		t.Errorf("zzz-empty should yield nil, got %v", got)
	}
}

// Empty query with kind filter exercises the general path.
func TestFixtureSourceSearchKindFilter(t *testing.T) {
	s := NewFixtureSource()
	m := s.Search("", Filters{Kind: "quote"}, 0)
	if len(m) == 0 {
		t.Fatal("expected at least one quote fixture")
	}
	for _, match := range m {
		if match.Card.Kind != cards.KindQuote {
			t.Errorf("non-quote match: %s", match.Card.Title)
		}
	}
}

// Empty query with a far-future Since filter returns nothing.
func TestFixtureSourceSearchSinceFuture(t *testing.T) {
	s := NewFixtureSource()
	far := time.Date(3000, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := s.Search("", Filters{Since: far}, 0); len(got) != 0 {
		t.Errorf("future Since should return empty, got %d matches", len(got))
	}
}

func TestFixtureSourceLastImportFalse(t *testing.T) {
	s := NewFixtureSource()
	if _, ok := s.LastImport(); ok {
		t.Error("fixture source should report no import")
	}
}

func TestFixtureSourceLastListSaveNoop(t *testing.T) {
	s := NewFixtureSource()
	if err := s.LastListSave(nil); err != nil {
		t.Errorf("LastListSave should be a no-op, got %v", err)
	}
}

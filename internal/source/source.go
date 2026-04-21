package source

import (
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
)

// Filters constrain Search results beyond the raw text query.
type Filters struct {
	Kind  string    // "article" | "image" | "quote" | "note" | ""
	From  string    // substring match against card.Source
	Since time.Time // zero value = unconstrained
	Tag   string    // exact-match tag filter
}

// Source provides the current card corpus to downstream command handlers.
// Phase 0 uses FixtureSource; Phase 1 adds a SQLite-backed implementation.
type Source interface {
	Count() int
	All() []cards.Card
	ByHandle(n int) (cards.Card, error)
	Search(query string, filters Filters, limit int) []render.Match
	LastImport() (time.Time, bool)
	LastListSave(matches []render.Match) error
}

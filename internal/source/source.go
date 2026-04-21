package source

import "github.com/samay58/cairn/internal/cards"

// Source provides the current card corpus to downstream command handlers.
// Phase 0 uses fixture-backed data; Phase 1 can swap in a real store.
type Source interface {
	Count() int
	All() []cards.Card
	ByHandle(int) (cards.Card, error)
}

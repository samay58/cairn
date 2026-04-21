package source

import (
	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/fixtures"
)

type FixtureSource struct{}

func NewFixtureSource() *FixtureSource {
	return &FixtureSource{}
}

func (f *FixtureSource) Count() int {
	return len(fixtures.All())
}

func (f *FixtureSource) All() []cards.Card {
	return fixtures.All()
}

func (f *FixtureSource) ByHandle(n int) (cards.Card, error) {
	return fixtures.ByHandle(n)
}

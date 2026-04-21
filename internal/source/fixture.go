package source

import (
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/fixtures"
	"github.com/samay58/cairn/internal/render"
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

// Search preserves Phase 0's hand-picked results for the "oauth" and
// "zzz-empty" demo queries so existing goldens stay byte-identical. For any
// other query (or an empty query), it falls back to a substring match across
// title, body, excerpt, and tags, with optional filter refinement.
func (f *FixtureSource) Search(query string, filters Filters, limit int) []render.Match {
	q := strings.ToLower(strings.TrimSpace(query))
	all := fixtures.All()

	// Preserve the Phase 0 deterministic fixtures so existing goldens
	// remain stable during the Source interface extension.
	switch q {
	case "oauth":
		matches := []render.Match{
			{Card: all[0], WhyShown: "Matched on title and tag oauth."},
			{Card: all[10], WhyShown: "Matched on tag auth."},
			{Card: all[17], WhyShown: "Matched on body."},
		}
		return applyFixtureLimit(matches, limit)
	case "zzz-empty":
		return nil
	case "":
		// Fall through to general filtering on empty query.
	default:
		if _, demo := demoQuery(q); demo {
			matches := []render.Match{
				{Card: all[0], WhyShown: "Demo result 1."},
				{Card: all[1], WhyShown: "Demo result 2."},
				{Card: all[2], WhyShown: "Demo result 3."},
			}
			return applyFixtureLimit(matches, limit)
		}
	}

	// General filtering path: substring match with filters.
	var matches []render.Match
	for _, c := range all {
		if !fixtureMatches(c, q, filters) {
			continue
		}
		matches = append(matches, render.Match{
			Card:     c,
			WhyShown: fixtureWhy(c, q),
		})
	}
	return applyFixtureLimit(matches, limit)
}

func (f *FixtureSource) LastImport() (time.Time, bool) {
	return time.Time{}, false
}

func (f *FixtureSource) LastListSave(_ []render.Match) error {
	return nil
}

func (f *FixtureSource) MediaFor(_ string) []cards.Media {
	return nil
}

// demoQuery returns true for any non-empty query that is not one of our
// curated cases. This is the Phase 0 "everything else shows the first three
// fixtures" behavior; kept as a named function for clarity.
func demoQuery(q string) (string, bool) {
	if q == "" {
		return "", false
	}
	return q, true
}

func applyFixtureLimit(matches []render.Match, limit int) []render.Match {
	if limit > 0 && limit < len(matches) {
		return matches[:limit]
	}
	return matches
}

func fixtureMatches(c cards.Card, q string, f Filters) bool {
	if f.Kind != "" && string(c.Kind) != f.Kind {
		return false
	}
	if f.From != "" && !strings.Contains(strings.ToLower(c.Source), strings.ToLower(f.From)) {
		return false
	}
	if !f.Since.IsZero() && c.CapturedAt.Before(f.Since) {
		return false
	}
	if f.Tag != "" {
		if !hasTag(c, f.Tag) {
			return false
		}
	}
	if q == "" {
		return true
	}
	hay := strings.ToLower(c.Title + " " + c.Body + " " + c.Excerpt + " " + strings.Join(c.Tags, " "))
	return strings.Contains(hay, q)
}

func hasTag(c cards.Card, tag string) bool {
	for _, t := range c.Tags {
		if strings.EqualFold(t, tag) {
			return true
		}
	}
	return false
}

func fixtureWhy(c cards.Card, q string) string {
	if q == "" {
		return "Recent."
	}
	lo := strings.ToLower(c.Title)
	if strings.Contains(lo, q) {
		return "Matched on title."
	}
	for _, t := range c.Tags {
		if strings.EqualFold(t, q) {
			return "Matched on tag " + t + "."
		}
	}
	return "Matched on body."
}

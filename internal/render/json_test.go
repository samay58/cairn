package render

import (
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/golden"
)

func sample() []Match {
	return []Match{
		{
			Card: cards.Card{
				Kind:       cards.KindArticle,
				Title:      "OAuth 2.0 Device Authorization Grant",
				Source:     "datatracker.ietf.org",
				Excerpt:    "Describes the OAuth 2.0 device authorization grant flow for browserless\nand input-constrained devices.",
				CapturedAt: time.Date(2026, 3, 14, 9, 22, 0, 0, time.UTC),
			},
			WhyShown: "matched on title and tag oauth",
		},
		{
			Card: cards.Card{
				Kind:       cards.KindQuote,
				Title:      "On craft",
				Source:     "Martha Beck",
				Body:       "The way you do anything is the way you do everything.",
				CapturedAt: time.Date(2026, 3, 18, 14, 5, 0, 0, time.UTC),
			},
			WhyShown: "matched on tag craft",
		},
	}
}

func TestCardListJSON(t *testing.T) {
	golden.Assert(t, "cardlist.json", CardListJSON(sample()))
}

func TestCardListJSONL(t *testing.T) {
	golden.Assert(t, "cardlist.jsonl", CardListJSONL(sample()))
}

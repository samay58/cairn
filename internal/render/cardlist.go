package render

import (
	"fmt"
	"strings"

	"github.com/samay58/cairn/internal/cards"
)

type Match struct {
	Card     cards.Card
	WhyShown string
}

type CardListItem struct {
	Handle   int        `json:"handle"`
	Card     cards.Card `json:"card"`
	WhyShown string     `json:"why_shown"`
}

// MetaLine formats the one-line type · source · date meta string.
// Source is omitted when empty.
func MetaLine(c cards.Card) string {
	parts := []string{c.Kind.Letter()}
	if c.Source != "" {
		parts = append(parts, c.Source)
	}
	parts = append(parts, c.CapturedAt.Format("2006-01-02"))
	return strings.Join(parts, TokenSeparator)
}

func CardList(matches []Match) string {
	var b strings.Builder
	for i, m := range matches {
		if i > 0 {
			b.WriteString("\n")
		}
		handle := fmt.Sprintf("@%d", i+1)
		fmt.Fprintf(&b, "%s  %s\n", handle, m.Card.Title)
		fmt.Fprintf(&b, "    %s\n", MetaLine(m.Card))
		for _, line := range WrapLines("    ", m.WhyShown, DefaultWidth) {
			fmt.Fprintf(&b, "%s\n", line)
		}
		for _, line := range WrapLines("    ", ExcerptText(m.Card), DefaultWidth) {
			fmt.Fprintf(&b, "%s\n", line)
		}
	}
	return b.String()
}

func CardListItems(matches []Match) []CardListItem {
	items := make([]CardListItem, 0, len(matches))
	for i, match := range matches {
		items = append(items, CardListItem{
			Handle:   i + 1,
			Card:     match.Card,
			WhyShown: match.WhyShown,
		})
	}
	return items
}

func ExcerptText(card cards.Card) string {
	switch {
	case card.Excerpt != "":
		return card.Excerpt
	case card.Body != "":
		return card.Body
	case card.Kind == cards.KindImage && card.Source != "":
		return "Saved image from " + card.Source + "."
	case card.Kind == cards.KindImage:
		return "Saved image."
	default:
		return "No excerpt available."
	}
}

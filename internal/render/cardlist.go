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
		excerpt := m.Card.Excerpt
		if excerpt == "" {
			excerpt = m.Card.Body
		}
		fmt.Fprintf(&b, "%s  %s\n", handle, m.Card.Title)
		fmt.Fprintf(&b, "    %s\n", MetaLine(m.Card))
		fmt.Fprintf(&b, "    %s\n", m.WhyShown)
		for _, line := range strings.Split(excerpt, "\n") {
			fmt.Fprintf(&b, "    %s\n", line)
		}
	}
	return b.String()
}

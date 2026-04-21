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

func CardList(matches []Match) string {
	var b strings.Builder
	for i, m := range matches {
		if i > 0 {
			b.WriteString("\n")
		}
		handle := fmt.Sprintf("@%d", i+1)
		metaParts := []string{m.Card.Kind.Letter()}
		if m.Card.Source != "" {
			metaParts = append(metaParts, m.Card.Source)
		}
		metaParts = append(metaParts, m.Card.CapturedAt.Format("2006-01-02"))
		meta := strings.Join(metaParts, TokenSeparator)
		excerpt := m.Card.Excerpt
		if excerpt == "" {
			excerpt = m.Card.Body
		}
		fmt.Fprintf(&b, "%s  %s\n", handle, m.Card.Title)
		fmt.Fprintf(&b, "    %s\n", meta)
		fmt.Fprintf(&b, "    %s\n", m.WhyShown)
		for _, line := range strings.Split(excerpt, "\n") {
			fmt.Fprintf(&b, "    %s\n", line)
		}
	}
	return b.String()
}

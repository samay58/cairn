package phoenix

import (
	"fmt"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

type MediaRef struct {
	Filename string
	RelPath  string
}

func RenderMarkdown(c cards.Card, media []MediaRef) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("mymind_id: ")
	b.WriteString(c.MyMindID)
	b.WriteString("\n")
	if c.URL != "" {
		b.WriteString("url: ")
		b.WriteString(c.URL)
		b.WriteString("\n")
	}
	b.WriteString("tags: ")
	b.WriteString(encodeTags(c.Tags))
	b.WriteString("\n")
	b.WriteString("captured_at: ")
	b.WriteString(c.CapturedAt.UTC().Format(time.RFC3339))
	b.WriteString("\n")
	b.WriteString("kind: ")
	b.WriteString(string(c.Kind))
	b.WriteString("\n---\n\n")

	b.WriteString("# ")
	b.WriteString(c.Title)
	b.WriteString("\n")

	body := strings.TrimSpace(c.Body)
	if body != "" {
		b.WriteString("\n")
		b.WriteString(body)
		b.WriteString("\n")
	}

	if len(media) > 0 {
		b.WriteString("\n## Attachments\n\n")
		for _, m := range media {
			fmt.Fprintf(&b, "- [%s](%s)\n", m.Filename, m.RelPath)
		}
	}
	return b.String()
}

func encodeTags(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}
	quoted := make([]string, len(tags))
	for i, t := range tags {
		quoted[i] = `"` + strings.ReplaceAll(t, `"`, `\"`) + `"`
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}


package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

// ParseCardsCSV reads a MyMind-style cards.csv. Column names match
// case-insensitively and accept synonyms (body ≡ text ≡ content). Rows missing
// id, kind, or title produce a warning and are skipped rather than failing the
// whole import.
func ParseCardsCSV(path string) ([]cards.Card, []string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}
	cols := normalizeHeader(header)

	var out []cards.Card
	var warnings []string
	lineNo := 1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		lineNo++
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("line %d: %v", lineNo, err))
			continue
		}
		c, ok, warn := rowToCard(cols, row)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("line %d: %s", lineNo, warn))
			continue
		}
		out = append(out, c)
	}
	return out, warnings, nil
}

// utf8BOM is the UTF-8 byte order mark that some exporters (including MyMind)
// prepend to the first line of CSV files.
const utf8BOM = "\xef\xbb\xbf"

func normalizeHeader(h []string) map[string]int {
	idx := map[string]int{}
	for i, name := range h {
		clean := strings.TrimPrefix(strings.TrimSpace(name), utf8BOM)
		idx[strings.ToLower(clean)] = i
	}
	return idx
}

func pick(cols map[string]int, row []string, names ...string) string {
	for _, n := range names {
		if i, ok := cols[n]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}

// kindAliases maps MyMind type values (lowercased, no spaces) to cairn Kind.
var kindAliases = map[string]cards.Kind{
	"article":      cards.KindArticle,
	"webpage":      cards.KindArticle,
	"document":     cards.KindArticle,
	"embed":        cards.KindArticle,
	"youtubevideo": cards.KindArticle,
	"link":         cards.KindArticle,
	"image":        cards.KindImage,
	"photo":        cards.KindImage,
	"quote":        cards.KindQuote,
	"note":         cards.KindNote,
}

func rowToCard(cols map[string]int, row []string) (cards.Card, bool, string) {
	id := pick(cols, row, "id", "mymind_id", "card_id")
	kindRaw := pick(cols, row, "type", "kind")
	title := pick(cols, row, "title")
	if id == "" || kindRaw == "" {
		return cards.Card{}, false, "missing id/type"
	}

	kindLower := strings.ToLower(strings.ReplaceAll(kindRaw, " ", ""))
	kind, err := cards.KindFromString(kindLower)
	if err != nil {
		if k, ok := kindAliases[kindLower]; ok {
			kind = k
		} else {
			return cards.Card{}, false, fmt.Sprintf("unknown kind %q", kindRaw)
		}
	}

	captured := pick(cols, row, "captured_at", "created_at", "created", "date")
	capturedAt, err := time.Parse(time.RFC3339, captured)
	if err != nil {
		capturedAt = time.Now().UTC()
	}
	tagsRaw := pick(cols, row, "tags")
	var tags []string
	if tagsRaw != "" {
		splitter := ";"
		if !strings.Contains(tagsRaw, ";") {
			splitter = ","
		}
		seen := map[string]bool{}
		for _, t := range strings.Split(tagsRaw, splitter) {
			if t = strings.TrimSpace(t); t != "" && !seen[t] {
				seen[t] = true
				tags = append(tags, t)
			}
		}
	}

	body := pick(cols, row, "body", "text", "content")
	note := pick(cols, row, "note")
	switch {
	case body == "" && note != "":
		body = note
	case body != "" && note != "":
		body = body + "\n\nNote: " + note
	}

	return cards.Card{
		ID:         id,
		MyMindID:   id,
		Kind:       kind,
		Title:      title,
		URL:        pick(cols, row, "url", "link"),
		Body:       body,
		Excerpt:    pick(cols, row, "excerpt", "description"),
		Source:     pick(cols, row, "source", "domain"),
		Tags:       tags,
		CapturedAt: capturedAt,
	}, true, ""
}

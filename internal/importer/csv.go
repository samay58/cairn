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

// Snapshot is a complete library read from one source (an export folder or
// the API), ready for ImportCards.
type Snapshot struct {
	Cards       []cards.Card
	RowsRead    int
	SkippedRows int
	Warnings    []string
}

// ParseCardsCSV reads a MyMind-style cards.csv. Column names match
// case-insensitively and accept synonyms (body ≡ text ≡ content). Rows missing
// id, kind, or title produce a warning and are skipped rather than failing the
// whole import.
func ParseCardsCSV(path string) ([]cards.Card, []string, error) {
	result, err := ParseCardsCSVDetailed(path)
	return result.Cards, result.Warnings, err
}

func ParseCardsCSVDetailed(path string) (Snapshot, error) {
	var result Snapshot
	f, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1

	header, err := r.Read()
	if err != nil {
		return result, fmt.Errorf("read header: %w", err)
	}
	cols := normalizeHeader(header)

	lineNo := 1
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		result.RowsRead++
		lineNo++
		if err != nil {
			result.SkippedRows++
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: %v", lineNo, err))
			continue
		}
		c, ok, warn := rowToCard(cols, row)
		if !ok {
			result.SkippedRows++
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: %s", lineNo, warn))
			continue
		}
		if warn != "" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("line %d: %s", lineNo, warn))
		}
		result.Cards = append(result.Cards, c)
	}
	return result, nil
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
	"article":             cards.KindArticle,
	"book":                cards.KindArticle,
	"business":            cards.KindArticle,
	"webpage":             cards.KindArticle,
	"document":            cards.KindArticle,
	"embed":               cards.KindArticle,
	"movie":               cards.KindArticle,
	"placeholder":         cards.KindArticle,
	"product":             cards.KindArticle,
	"redditpost":          cards.KindArticle,
	"repository":          cards.KindArticle,
	"softwareapplication": cards.KindArticle,
	"tvseries":            cards.KindArticle,
	"video":               cards.KindArticle,
	"videogame":           cards.KindArticle,
	"wikipediaarticle":    cards.KindArticle,
	"xpost":               cards.KindArticle,
	"youtubevideo":        cards.KindArticle,
	"link":                cards.KindArticle,
	"image":               cards.KindImage,
	"photo":               cards.KindImage,
	"quote":               cards.KindQuote,
	"note":                cards.KindNote,
}

// ResolveKind maps a MyMind type name ("WebPage", "XPost", ...) to a cairn
// Kind. Unknown names fall back to article and report known=false.
func ResolveKind(raw string) (kind cards.Kind, known bool) {
	lower := strings.ToLower(strings.ReplaceAll(raw, " ", ""))
	if k, err := cards.KindFromString(lower); err == nil {
		return k, true
	}
	if k, ok := kindAliases[lower]; ok {
		return k, true
	}
	return cards.KindArticle, false
}

func rowToCard(cols map[string]int, row []string) (cards.Card, bool, string) {
	id := pick(cols, row, "id", "mymind_id", "card_id")
	kindRaw := pick(cols, row, "type", "kind")
	title := pick(cols, row, "title")
	if id == "" || kindRaw == "" {
		return cards.Card{}, false, "missing id/type"
	}

	kind, known := ResolveKind(kindRaw)
	fellBackKind := !known

	captured := pick(cols, row, "captured_at", "created_at", "created", "date")
	capturedAt, err := time.Parse(time.RFC3339, captured)
	if err != nil {
		return cards.Card{}, false, fmt.Sprintf("invalid created %q", captured)
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

	warn := ""
	if fellBackKind {
		warn = fmt.Sprintf("unknown kind %q treated as article", kindRaw)
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
	}, true, warn
}

package mymindapi

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/importer"
)

// Object is the subset of a MyMind API object cairn uses.
type Object struct {
	ID         string     `json:"id"`
	Title      string     `json:"title"`
	Summary    string     `json:"summary"`
	EntityType string     `json:"entityType"`
	MainEntity mainEntity `json:"mainEntity"`
	Content    *prose     `json:"content"`
	Notes      []note     `json:"notes"`
	Tags       []tag      `json:"tags"`
	Source     struct {
		URL string `json:"url"`
	} `json:"source"`
	Blob    *blobRef `json:"blob"`
	Created string   `json:"created"`
}

type mainEntity struct {
	Title       string `json:"title"`
	Publication struct {
		Name string `json:"name"`
	} `json:"publication"`
}

type note struct {
	Content prose `json:"content"`
}

type tag struct {
	Name string `json:"name"`
}

type blobRef struct {
	Type string `json:"type"`
}

// prose is MyMind's rich text: application/prose+json holds a ProseMirror
// document; other types hold a plain string.
type prose struct {
	Type string          `json:"type"`
	Body json.RawMessage `json:"body"`
}

type proseNode struct {
	Type    string      `json:"type"`
	Text    string      `json:"text"`
	Content []proseNode `json:"content"`
}

// Text renders prose as plain text, one paragraph per block.
func (p prose) Text() string {
	var s string
	if json.Unmarshal(p.Body, &s) == nil {
		return strings.TrimSpace(s)
	}
	var doc proseNode
	if json.Unmarshal(p.Body, &doc) != nil {
		return ""
	}
	var blocks []string
	for _, block := range doc.Content {
		if t := strings.TrimSpace(inlineText(block)); t != "" {
			blocks = append(blocks, t)
		}
	}
	return strings.Join(blocks, "\n\n")
}

func inlineText(n proseNode) string {
	if n.Type == "text" {
		return n.Text
	}
	if n.Type == "hardBreak" {
		return "\n"
	}
	var b strings.Builder
	for i, c := range n.Content {
		if i > 0 && isBlock(c.Type) {
			b.WriteString("\n")
		}
		b.WriteString(inlineText(c))
	}
	return b.String()
}

func isBlock(t string) bool {
	return t == "paragraph" || t == "listItem" || t == "heading" || t == "blockquote"
}

// CardID converts an API id ("00000000MDE0O4mm...") to the id MyMind's CSV
// export uses ("MDE0O4mm..."), so both paths address the same card.
func CardID(apiID string) string {
	return strings.TrimPrefix(apiID, "00000000")
}

var wikilink = regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

// plainSummary drops MyMind's [[entity]] markup. In an Obsidian vault each
// one would be a link to a note that does not exist.
func plainSummary(s string) string {
	return strings.TrimSpace(wikilink.ReplaceAllStringFunc(s, func(m string) string {
		parts := wikilink.FindStringSubmatch(m)
		if parts[2] != "" {
			return parts[2]
		}
		return parts[1]
	}))
}

// cleanURL removes utm_* tracking parameters, matching what the CSV export
// already strips.
func cleanURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.RawQuery == "" {
		return raw
	}
	q := u.Query()
	for k := range q {
		if strings.HasPrefix(strings.ToLower(k), "utm_") {
			q.Del(k)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// ToCard maps an object to a cairn card. The body is the object's own text
// (notes and other text objects) or MyMind's AI summary, followed by Samay's
// notes in the CSV path's "Note:" form. warn is non-empty for an unknown type.
func (o Object) ToCard() (c cards.Card, warn string, err error) {
	created, err := time.Parse(time.RFC3339Nano, o.Created)
	if err != nil {
		return cards.Card{}, "", fmt.Errorf("object %s: invalid created %q", o.ID, o.Created)
	}
	kind, known := importer.ResolveKind(o.EntityType)
	if !known {
		warn = fmt.Sprintf("unknown type %q treated as article", o.EntityType)
	}
	title := o.Title
	if title == "" {
		title = o.MainEntity.Title
	}
	body := plainSummary(o.Summary)
	if o.Content != nil {
		if t := o.Content.Text(); t != "" {
			body = t
		}
	}
	var notes []string
	for _, n := range o.Notes {
		if t := n.Content.Text(); t != "" {
			notes = append(notes, t)
		}
	}
	if len(notes) > 0 {
		joined := strings.Join(notes, "\n\n")
		if body == "" {
			body = joined
		} else {
			body += "\n\nNote: " + joined
		}
	}
	var tags []string
	seen := map[string]bool{}
	for _, t := range o.Tags {
		if name := strings.TrimSpace(t.Name); name != "" && !seen[name] {
			seen[name] = true
			tags = append(tags, name)
		}
	}
	id := CardID(o.ID)
	return cards.Card{
		ID:         id,
		MyMindID:   id,
		Kind:       kind,
		Title:      title,
		URL:        cleanURL(o.Source.URL),
		Body:       body,
		Source:     o.MainEntity.Publication.Name,
		Tags:       tags,
		CapturedAt: created.UTC().Truncate(time.Second),
	}, warn, nil
}

// BlobExtension picks a file extension for an attachment's media type.
func BlobExtension(mime string) string {
	mime, _, _ = strings.Cut(mime, ";")
	switch strings.TrimSpace(strings.ToLower(mime)) {
	case "application/pdf":
		return "pdf"
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "video/mp4":
		return "mp4"
	}
	return "bin"
}

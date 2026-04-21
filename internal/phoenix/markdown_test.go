package phoenix

import (
	"strings"
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

func TestRenderMarkdownBasicArticle(t *testing.T) {
	c := cards.Card{
		ID:         "row1",
		MyMindID:   "MDE0O3xIKzJh4Y",
		Kind:       cards.KindArticle,
		Title:      "How AI web search works",
		URL:        "https://example.com/post",
		Body:       "First paragraph.\n\nSecond paragraph.",
		Tags:       []string{"ai", "search"},
		CapturedAt: time.Date(2026, 4, 20, 20, 55, 33, 0, time.UTC),
	}
	got := RenderMarkdown(c, nil)
	want := "---\n" +
		"mymind_id: MDE0O3xIKzJh4Y\n" +
		"url: https://example.com/post\n" +
		"tags: [\"ai\", \"search\"]\n" +
		"captured_at: 2026-04-20T20:55:33Z\n" +
		"kind: article\n" +
		"---\n\n" +
		"# How AI web search works\n\n" +
		"First paragraph.\n\n" +
		"Second paragraph.\n"
	if got != want {
		t.Fatalf("markdown mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestRenderMarkdownWithMedia(t *testing.T) {
	c := cards.Card{
		MyMindID:   "MDE0O3xIKzJh4Y",
		Kind:       cards.KindArticle,
		Title:      "With attachment",
		Body:       "hello",
		CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
	}
	media := []MediaRef{
		{Filename: "MDE0O3xIKzJh4Y.pdf", RelPath: "_media/3f/5a/3f5a.pdf"},
	}
	got := RenderMarkdown(c, media)
	if !strings.Contains(got, "## Attachments") {
		t.Fatalf("missing Attachments section")
	}
	if !strings.Contains(got, "[MDE0O3xIKzJh4Y.pdf](_media/3f/5a/3f5a.pdf)") {
		t.Fatalf("missing attachment link")
	}
}

func TestRenderMarkdownEmptyBody(t *testing.T) {
	c := cards.Card{MyMindID: "a", Kind: cards.KindNote, Title: "t",
		CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)}
	got := RenderMarkdown(c, nil)
	if !strings.Contains(got, "# t\n") {
		t.Fatalf("expected title heading")
	}
	if strings.Contains(got, "\n\n\n") {
		t.Fatalf("unexpected triple newline in empty-body output:\n%s", got)
	}
}

package phoenix

import (
	"testing"
	"time"
)

func TestSlug(t *testing.T) {
	cases := map[string]string{
		"How AI web search works":        "how-ai-web-search-works",
		"Claude & the meaning of life":   "claude-the-meaning-of-life",
		"   leading / trailing   ":       "leading-trailing",
		"éléphant":                       "elephant",
		"":                               "untitled",
		"A/B Testing with 200% coverage": "a-b-testing-with-200-coverage",
	}
	for in, want := range cases {
		got := Slug(in)
		if got != want {
			t.Errorf("Slug(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDailyFilename(t *testing.T) {
	ts := time.Date(2026, 4, 20, 20, 55, 33, 0, time.UTC)
	got := DailyFilename(ts, "How AI web search works")
	want := "2026-04-20-how-ai-web-search-works.md"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestDailyFilenameTruncatesLongSlugs(t *testing.T) {
	ts := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	long := "A very long title that keeps on going and going past the truncation threshold we set"
	got := DailyFilename(ts, long)
	if len(got) > 74 {
		t.Fatalf("filename too long: %q (%d)", got, len(got))
	}
}

func TestCollisionSuffix(t *testing.T) {
	exists := map[string]bool{
		"2026-04-20-hello.md":   true,
		"2026-04-20-hello-2.md": true,
	}
	got := UniqueFilename("2026-04-20-hello.md", func(name string) bool { return exists[name] })
	want := "2026-04-20-hello-3.md"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCollisionSuffixNoCollision(t *testing.T) {
	got := UniqueFilename("x.md", func(string) bool { return false })
	if got != "x.md" {
		t.Fatalf("got %q, want x.md", got)
	}
}

func TestMediaPath(t *testing.T) {
	got := MediaRelPath("3f5abc0000000000000000000000000000000000000000000000000000000000", "pdf")
	want := "_media/3f/5a/3f5abc0000000000000000000000000000000000000000000000000000000000.pdf"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestRelMediaLinkFromCard(t *testing.T) {
	got := RelMediaLink("_media/3f/5a/3f5a.pdf")
	want := "_media/3f/5a/3f5a.pdf"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

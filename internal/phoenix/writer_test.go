package phoenix

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

func readDir(t *testing.T, p string) []string {
	t.Helper()
	entries, err := os.ReadDir(p)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func contentSHA(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func TestWriterDryRunWritesNothing(t *testing.T) {
	tmp := t.TempDir()
	w := &Writer{Root: tmp, DryRun: true}
	bundles := []CardBundle{{
		Card: cards.Card{
			ID: "r1", MyMindID: "m1", Kind: cards.KindArticle,
			Title: "T", Body: "body",
			CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		},
	}}
	report, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if report.CardsWritten != 1 {
		t.Errorf("report.CardsWritten = %d, want 1", report.CardsWritten)
	}
	entries := readDir(t, tmp)
	if len(entries) != 0 {
		t.Errorf("dry-run wrote %d entries, want 0", len(entries))
	}
	_ = bytes.Buffer{}
	_ = filepath.Join
}

func TestWriterWritesCardMarkdown(t *testing.T) {
	tmp := t.TempDir()
	w := &Writer{Root: tmp}
	bundles := []CardBundle{{
		Card: cards.Card{
			MyMindID:   "m1",
			Kind:       cards.KindArticle,
			Title:      "Hello world",
			Body:       "Para one.\n\nPara two.",
			CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		},
	}}
	r, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if r.CardsWritten != 1 {
		t.Fatalf("CardsWritten = %d", r.CardsWritten)
	}
	p := filepath.Join(tmp, "2026-04-20-hello-world.md")
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(got, []byte("# Hello world")) {
		t.Fatalf("missing title in %s", got)
	}
	if !bytes.Contains(got, []byte("mymind_id: m1")) {
		t.Fatalf("missing frontmatter id")
	}
}

func TestWriterIdempotent(t *testing.T) {
	tmp := t.TempDir()
	w := &Writer{Root: tmp}
	bundles := []CardBundle{{
		Card: cards.Card{
			MyMindID: "m1", Kind: cards.KindNote, Title: "T", Body: "x",
			CapturedAt: time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC),
		},
	}}
	if _, err := w.Write(bundles); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(tmp, "2026-04-20-t.md")
	firstStat, _ := os.Stat(p)
	time.Sleep(10 * time.Millisecond)
	stored, _ := os.ReadFile(p)
	want := RenderMarkdown(bundles[0].Card, nil)
	if string(stored) != want {
		t.Fatalf("stored != rendered\nstored: %q\nwant:   %q", stored, want)
	}
	r, err := w.Write(bundles)
	if err != nil {
		t.Fatal(err)
	}
	if r.CardsUnchanged != 1 || r.CardsWritten != 0 {
		t.Fatalf("expected Unchanged=1 Written=0, got Written=%d Unchanged=%d", r.CardsWritten, r.CardsUnchanged)
	}
	secondStat, _ := os.Stat(p)
	if !secondStat.ModTime().Equal(firstStat.ModTime()) {
		t.Errorf("mtime changed on idempotent rewrite")
	}
}

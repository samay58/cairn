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

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/samay58/cairn/internal/source"
	"github.com/samay58/cairn/internal/storage/sqlite"
)

func TestOpenSourceFallsBackToFixtureWhenNoDB(t *testing.T) {
	s, err := openSource(filepath.Join(t.TempDir(), "nope.db"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*source.FixtureSource); !ok {
		t.Errorf("expected *source.FixtureSource, got %T", s)
	}
}

func TestOpenSourceReturnsSQLiteWhenDBExists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cairn.db")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := openSource(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.(*sqlite.SQLiteSource); !ok {
		t.Errorf("expected *sqlite.SQLiteSource, got %T", s)
	}
}

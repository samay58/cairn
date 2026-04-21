package source_test

import (
	"testing"

	"github.com/samay58/cairn/internal/source"
)

func TestFixtureSourceMediaForReturnsEmpty(t *testing.T) {
	src := source.NewFixtureSource()
	got := src.MediaFor("anything")
	if len(got) != 0 {
		t.Fatalf("expected empty, got %d", len(got))
	}
}

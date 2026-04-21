package importer

import (
	"strings"
	"testing"
)

func TestChunkShortBodyReturnsOne(t *testing.T) {
	chunks := Chunk("A short body.")
	if len(chunks) != 1 {
		t.Fatalf("got %d chunks, want 1", len(chunks))
	}
	if chunks[0].Text != "A short body." {
		t.Errorf("text = %q", chunks[0].Text)
	}
}

func TestChunkLongBodySplitsOnParagraphs(t *testing.T) {
	// Each paragraph is 11 words; 80 repeats = 880 words total.
	// The chunker flushes when bufWords >= 200 and adding next para would exceed 600,
	// so we get at least two chunks.
	long := strings.Repeat("paragraph one word word word word word word word word word.\n\n", 80)
	chunks := Chunk(long)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks for long body, got %d", len(chunks))
	}
	for i, ch := range chunks {
		words := len(strings.Fields(ch.Text))
		if words > 700 {
			t.Errorf("chunk %d has %d words, want <=700", i, words)
		}
	}
}

func TestChunkChecksumsStable(t *testing.T) {
	body := "Once more with feeling.\n\nSecond paragraph."
	a := Chunk(body)
	b := Chunk(body)
	if len(a) != len(b) {
		t.Fatalf("chunk counts differ: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].Checksum != b[i].Checksum {
			t.Errorf("chunk %d checksum differs between runs", i)
		}
	}
}

func TestChunkEmptyBodyReturnsNil(t *testing.T) {
	if got := Chunk(""); got != nil {
		t.Errorf("Chunk('') = %v, want nil", got)
	}
	if got := Chunk("   "); got != nil {
		t.Errorf("Chunk('   ') = %v, want nil", got)
	}
}

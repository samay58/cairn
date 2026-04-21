package fixtures

import (
	"testing"

	"github.com/samay58/cairn/internal/cards"
)

func TestAllReturnsTwentyFive(t *testing.T) {
	got := All()
	if len(got) != 25 {
		t.Fatalf("len(All()) = %d, want 25", len(got))
	}
}

func TestAllCoversEveryKind(t *testing.T) {
	kinds := map[cards.Kind]int{}
	for _, c := range All() {
		kinds[c.Kind]++
	}
	for _, k := range []cards.Kind{cards.KindArticle, cards.KindImage, cards.KindQuote, cards.KindNote} {
		if kinds[k] == 0 {
			t.Errorf("no fixture cards of kind %q", k)
		}
	}
}

func TestByHandleIsOneBased(t *testing.T) {
	c, err := ByHandle(1)
	if err != nil {
		t.Fatal(err)
	}
	if c.ID != All()[0].ID {
		t.Errorf("ByHandle(1) != All()[0]")
	}
	if _, err := ByHandle(0); err == nil {
		t.Error("expected error for handle 0")
	}
	if _, err := ByHandle(26); err == nil {
		t.Error("expected error for handle out of range")
	}
}

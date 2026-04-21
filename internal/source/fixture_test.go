package source

import "testing"

func TestFixtureSourceCount(t *testing.T) {
	src := NewFixtureSource()
	if got := src.Count(); got != 25 {
		t.Fatalf("Count() = %d, want 25", got)
	}
}

func TestFixtureSourceByHandle(t *testing.T) {
	src := NewFixtureSource()
	card, err := src.ByHandle(1)
	if err != nil {
		t.Fatal(err)
	}
	if card.ID != "c_0001" {
		t.Fatalf("ByHandle(1) id = %q, want c_0001", card.ID)
	}
}

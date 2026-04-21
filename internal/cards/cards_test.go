package cards

import "testing"

func TestKindLetter(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindArticle, "a"},
		{KindImage, "i"},
		{KindQuote, "q"},
		{KindNote, "n"},
	}
	for _, tc := range tests {
		if got := tc.kind.Letter(); got != tc.want {
			t.Errorf("Kind(%q).Letter() = %q, want %q", tc.kind, got, tc.want)
		}
	}
}

func TestKindFromString(t *testing.T) {
	k, err := KindFromString("article")
	if err != nil {
		t.Fatal(err)
	}
	if k != KindArticle {
		t.Errorf("got %q, want %q", k, KindArticle)
	}
	if _, err := KindFromString("bogus"); err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

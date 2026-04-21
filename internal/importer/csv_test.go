package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCardsCSVHappyPath(t *testing.T) {
	got, warnings, err := ParseCardsCSV(filepath.Join("..", "..", "testdata", "mymind_sample_export", "cards.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Errorf("parsed %d cards, want 4", len(got))
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	first := got[0]
	if first.MyMindID != "mm_1" || first.Title != "Deep Work" {
		t.Errorf("first card: %+v", first)
	}
	if len(first.Tags) != 2 {
		t.Errorf("first card tags = %v, want 2 entries", first.Tags)
	}
}

func TestParseCardsCSVMalformedRowProducesWarning(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.csv")
	body := "id,type,title\nmm_1,article,Good\n,,\nmm_3,,Missing type\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, warnings, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("got %d valid cards, want 1 (Good)", len(got))
	}
	if len(warnings) == 0 {
		t.Error("expected warnings for malformed rows")
	}
}

func TestParseCardsCSVColumnSynonyms(t *testing.T) {
	path := filepath.Join(t.TempDir(), "syns.csv")
	body := "ID,Kind,Title,Link,Content,Domain,Tags,Date\nmm_x,article,Widget,https://example.com,Body text,example.com,one;two,2026-04-10T00:00:00Z\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d, want 1", len(got))
	}
	if got[0].URL != "https://example.com" || got[0].Body != "Body text" || got[0].Source != "example.com" {
		t.Errorf("synonym mapping wrong: %+v", got[0])
	}
}

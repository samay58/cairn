package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/cards"
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
	body := "id,type,title,created\nmm_1,article,Good,2026-04-01T00:00:00Z\n,,,\nmm_3,,Missing type,2026-04-01T00:00:00Z\n"
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

func TestParseCardsCSVStripsBOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bom.csv")
	body := "\xef\xbb\xbfid,type,title,url,content,note,tags,created\nmm_1,Article,Hello,,The body.,,design,2026-04-01T10:00:00Z\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, warnings, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1 (BOM mis-parse likely dropped the row). warnings=%v", len(got), warnings)
	}
	if got[0].ID != "mm_1" || got[0].Title != "Hello" || got[0].Kind != cards.KindArticle {
		t.Errorf("parsed row wrong: %+v", got[0])
	}
}

func TestParseCardsCSVRealMyMindTypes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "real.csv")
	body := "\xef\xbb\xbfid,type,title,url,content,note,tags,created\n" +
		"a,Article,Web article,https://a.com,Body,,tag,2026-04-01T00:00:00Z\n" +
		"w,WebPage,Web page,https://w.com,,,tag,2026-04-01T00:00:00Z\n" +
		"d,Document,PDF,,,,doc,2026-04-01T00:00:00Z\n" +
		"e,Embed,Tweet,https://x.com/t,,,tag,2026-04-01T00:00:00Z\n" +
		"y,YouTubeVideo,Vid,https://yt.com,,,tag,2026-04-01T00:00:00Z\n" +
		"n,Note,Thought,,Body text,,tag,2026-04-01T00:00:00Z\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("got %d cards, want 6", len(got))
	}
	kinds := map[cards.Kind]int{}
	for _, c := range got {
		kinds[c.Kind]++
	}
	if kinds[cards.KindArticle] != 5 {
		t.Errorf("expected 5 article-aliased cards, got %d (distribution %v)", kinds[cards.KindArticle], kinds)
	}
	if kinds[cards.KindNote] != 1 {
		t.Errorf("expected 1 note card, got %d", kinds[cards.KindNote])
	}
}

func TestParseCardsCSVNoteFallsBackToBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "note.csv")
	body := "\xef\xbb\xbfid,type,title,url,content,note,tags,created\nmm_n,Note,My note,,,\"private annotation text\",tag,2026-04-01T00:00:00Z\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1", len(got))
	}
	if got[0].Body == "" {
		t.Errorf("expected note to populate body, got empty")
	}
	if !strings.Contains(got[0].Body, "private annotation text") {
		t.Errorf("body missing note content: %q", got[0].Body)
	}
}

func TestParseCardsCSVEmptyTitleNoteAccepted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "untitled.csv")
	body := "\xef\xbb\xbfid,type,title,url,content,note,tags,created\n" +
		"mm_u,Note,,,\"my first note\",,tag,2026-04-01T00:00:00Z\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, _, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d cards, want 1 (empty-title note should be accepted)", len(got))
	}
	if got[0].Kind != cards.KindNote {
		t.Errorf("expected KindNote, got %v", got[0].Kind)
	}
	if !strings.Contains(got[0].Body, "my first note") {
		t.Errorf("body missing note content: %q", got[0].Body)
	}
}

func TestParseCardsCSVInvalidCreatedWarnsAndSkips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad-created.csv")
	body := "\xef\xbb\xbfid,type,title,url,content,note,tags,created\n" +
		"mm_bad,Article,Hello,,Body,,tag,not-a-time\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, warnings, err := ParseCardsCSV(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got %d cards, want 0", len(got))
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "invalid created") {
		t.Fatalf("expected invalid created warning, got %v", warnings)
	}
}

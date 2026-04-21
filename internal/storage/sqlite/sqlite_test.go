package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	_ "modernc.org/sqlite"
)

// seed creates an in-file DB with migrations applied and two cards inserted,
// plus one sync_log entry, and one tag on card c_1.
func seed(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cairn.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO cards(id, mymind_id, kind, title, body, captured_at, updated_at) VALUES
		('c_1','mm_1','article','Deep Work','Rules for focused success.',?,?),
		('c_2','mm_2','quote','On craft','The way you do anything is the way you do everything.',?,?)`,
		now, now, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO tags(card_id, tag) VALUES ('c_1','focus'), ('c_1','productivity')`); err != nil {
		t.Fatal(err)
	}
	var c1RowID int64
	if err := db.QueryRow(`SELECT rowid FROM cards WHERE id = 'c_1'`).Scan(&c1RowID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cards_fts(cards_fts, rowid, title, body, tags_flat) VALUES ('delete', ?, ?, ?, '')`,
		c1RowID, "Deep Work", "Rules for focused success."); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cards_fts(rowid, title, body, tags_flat) VALUES (?, ?, ?, ?)`,
		c1RowID, "Deep Work", "Rules for focused success.", "focus productivity"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO sync_log(started_at, finished_at, delta_count, status) VALUES (?,?,2,'ok')`, now, now); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSQLiteSourceCount(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}
	if got := s.Count(); got != 2 {
		t.Errorf("Count() = %d, want 2", got)
	}
}

func TestSQLiteSourceLastImport(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}
	ts, ok := s.LastImport()
	if !ok {
		t.Fatal("LastImport should return ok=true after seed")
	}
	if time.Since(ts) > time.Minute {
		t.Errorf("LastImport too stale: %v", ts)
	}
}

func TestSQLiteSourceAllIncludesTags(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}
	got := s.All()
	if len(got) != 2 {
		t.Fatalf("All() len = %d, want 2", len(got))
	}
	var c1 cards.Card
	for _, c := range got {
		if c.ID == "c_1" {
			c1 = c
		}
	}
	if len(c1.Tags) != 2 {
		t.Errorf("expected 2 tags on c_1, got %v", c1.Tags)
	}
}

func TestSQLiteSourceSearchFTS(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}
	matches := s.Search("craft", source.Filters{}, 0)
	if len(matches) == 0 {
		t.Fatal("expected match for 'craft'")
	}
	if matches[0].Card.Title != "On craft" {
		t.Errorf("top hit = %q, want 'On craft'", matches[0].Card.Title)
	}
	if matches[0].WhyShown == "" {
		t.Error("WhyShown should not be empty")
	}
}

func TestSQLiteSourceSearchKindFilter(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}
	matches := s.Search("", source.Filters{Kind: "quote"}, 0)
	for _, m := range matches {
		if m.Card.Kind != "quote" {
			t.Errorf("non-quote match slipped through: %s", m.Card.Kind)
		}
	}
}

func TestSQLiteSourceSearchLimit(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}
	matches := s.Search("", source.Filters{}, 1)
	if len(matches) != 1 {
		t.Errorf("limit=1 returned %d matches", len(matches))
	}
}

func TestSQLiteSourceSearchTagWhyShown(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}
	matches := s.Search("productivity", source.Filters{}, 0)
	if len(matches) != 1 {
		t.Fatalf("expected one tag match, got %d", len(matches))
	}
	if matches[0].WhyShown != "Matched on tag productivity." {
		t.Fatalf("why shown = %q", matches[0].WhyShown)
	}
}

func TestSQLiteSourceHandlesRoundTrip(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}

	matches := []render.Match{
		{Card: cards.Card{ID: "c_1"}},
		{Card: cards.Card{ID: "c_2"}},
	}
	if err := s.LastListSave(matches); err != nil {
		t.Fatal(err)
	}
	c2, err := s.ByHandle(2)
	if err != nil {
		t.Fatal(err)
	}
	if c2.ID != "c_2" {
		t.Errorf("@2 = %q, want c_2", c2.ID)
	}
	if _, err := s.ByHandle(99); err == nil {
		t.Error("@99 should error")
	}

	// A new save clears the old table.
	single := []render.Match{{Card: cards.Card{ID: "c_1"}}}
	if err := s.LastListSave(single); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ByHandle(2); err == nil {
		t.Error("after single-item save, @2 should error")
	}
}

func TestSQLiteSourceByHandleIncludesTags(t *testing.T) {
	db := seed(t)
	defer db.Close()
	s := &SQLiteSource{DB: db}

	if err := s.LastListSave([]render.Match{{Card: cards.Card{ID: "c_1"}}}); err != nil {
		t.Fatal(err)
	}
	card, err := s.ByHandle(1)
	if err != nil {
		t.Fatal(err)
	}
	if len(card.Tags) != 2 {
		t.Fatalf("tags = %v, want 2 tags", card.Tags)
	}
}

func TestOpenSetsDesktopPragmas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cairn.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	var journal string
	if err := s.DB.QueryRow(`PRAGMA journal_mode`).Scan(&journal); err != nil {
		t.Fatal(err)
	}
	if journal != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journal)
	}

	var synchronous int
	if err := s.DB.QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatal(err)
	}
	if synchronous != 1 {
		t.Fatalf("synchronous = %d, want 1 (NORMAL)", synchronous)
	}
}

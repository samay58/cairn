package sqlite

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestMigrateAppliesSchema0001(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}

	var version int
	if err := db.QueryRow("SELECT max(version) FROM schema_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version < 1 {
		t.Errorf("schema_version = %d, want >= 1", version)
	}

	for _, name := range []string{"cards", "card_meta", "tags", "media", "chunks", "sync_log", "handles", "cards_fts"} {
		var got string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE name=?", name).Scan(&got)
		if err != nil {
			t.Errorf("table %q missing: %v", name, err)
		}
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("second Migrate should succeed: %v", err)
	}
}

func TestMigration0002RemovesEmptyCardIDMedia(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Simulate a Phase 1 database: apply migrations through Migrate, then
	// roll schema_version back to 1 so 0002 is re-run and its effect is
	// observable. In production 0002 runs once on the first Phase 2a open.
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM schema_version WHERE version = 2`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO cards(id, mymind_id, kind, title, captured_at, updated_at)
		VALUES ('r1','m1','article','t','2026-04-20T00:00:00Z','2026-04-20T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO media(card_id, kind, path, sha256, mime)
		VALUES ('','other','/legacy','sha','x/y')`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(db); err != nil {
		t.Fatal(err)
	}
	var n int
	db.QueryRow(`SELECT count(*) FROM media WHERE card_id=''`).Scan(&n)
	if n != 0 {
		t.Fatalf("expected 0 empty card_id rows after 0002, got %d", n)
	}
	var v int
	db.QueryRow(`SELECT max(version) FROM schema_version`).Scan(&v)
	if v < 2 {
		t.Fatalf("schema_version = %d, want >= 2", v)
	}
}

func TestOpenEnablesForeignKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.db")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var on int
	s.DB.QueryRow("PRAGMA foreign_keys").Scan(&on)
	if on != 1 {
		t.Fatalf("PRAGMA foreign_keys = %d, want 1", on)
	}
}

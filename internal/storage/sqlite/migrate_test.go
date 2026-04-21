package sqlite

import (
	"database/sql"
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
	if version != 1 {
		t.Errorf("schema_version = %d, want 1", version)
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

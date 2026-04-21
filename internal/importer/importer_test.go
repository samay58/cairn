package importer

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/samay58/cairn/internal/storage/sqlite"
	_ "modernc.org/sqlite"
)

func mustOpen(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "cairn.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := sqlite.Migrate(db); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestImportEndToEndSampleExport(t *testing.T) {
	db := mustOpen(t)
	defer db.Close()

	result, err := Import(db, filepath.Join("..", "..", "testdata", "mymind_sample_export"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Inserted != 4 {
		t.Errorf("inserted %d, want 4", result.Inserted)
	}
	if result.MediaCount != 1 {
		t.Errorf("media %d, want 1", result.MediaCount)
	}

	var status string
	var finished sql.NullString
	if err := db.QueryRow(`SELECT status, finished_at FROM sync_log ORDER BY id DESC LIMIT 1`).Scan(&status, &finished); err != nil {
		t.Fatal(err)
	}
	if status != "ok" {
		t.Errorf("status %q, want ok", status)
	}
	if !finished.Valid {
		t.Error("finished_at should be set")
	}

	var ftsCount int
	if err := db.QueryRow(`SELECT count(*) FROM cards_fts`).Scan(&ftsCount); err != nil {
		t.Fatal(err)
	}
	if ftsCount != 4 {
		t.Errorf("cards_fts rows = %d, want 4", ftsCount)
	}

	var chunkCount int
	if err := db.QueryRow(`SELECT count(*) FROM chunks`).Scan(&chunkCount); err != nil {
		t.Fatal(err)
	}
	if chunkCount < 2 {
		t.Errorf("chunks = %d, want >= 2", chunkCount)
	}
}

func TestImportTombstonesMissingCards(t *testing.T) {
	db := mustOpen(t)
	defer db.Close()

	// First import — everything fresh.
	if _, err := Import(db, filepath.Join("..", "..", "testdata", "mymind_sample_export")); err != nil {
		t.Fatal(err)
	}

	// Write a smaller export to a tmp dir (only mm_1).
	tmp := t.TempDir()
	small := "id,type,title,url,body,excerpt,source,tags,captured_at\nmm_1,article,Deep Work,https://cal.newport.com/books/deep-work/,Rules for focused success in a distracted world.,,cal.newport.com,\"focus;productivity\",2026-03-01T09:00:00Z\n"
	if err := writeFile(filepath.Join(tmp, "cards.csv"), small); err != nil {
		t.Fatal(err)
	}

	r, err := Import(db, tmp)
	if err != nil {
		t.Fatal(err)
	}
	if r.Tombstoned != 3 {
		t.Errorf("tombstoned %d, want 3 (mm_2, mm_3, mm_4)", r.Tombstoned)
	}
}

func TestImportHardDeletesAfter30Days(t *testing.T) {
	db := mustOpen(t)
	defer db.Close()

	old := time.Now().UTC().Add(-31 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO cards(id, mymind_id, kind, title, captured_at, updated_at, deleted_at) VALUES ('c_old','mm_old','article','Stale',?,?,?)`, old, old, old); err != nil {
		t.Fatal(err)
	}

	if _, err := Import(db, filepath.Join("..", "..", "testdata", "mymind_sample_export")); err != nil {
		t.Fatal(err)
	}

	var n int
	if err := db.QueryRow(`SELECT count(*) FROM cards WHERE id = 'c_old'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("stale card still present, count = %d", n)
	}
}

func TestImportMissingDirReturnsError(t *testing.T) {
	db := mustOpen(t)
	defer db.Close()
	if _, err := Import(db, "/tmp/does-not-exist-xyz"); err == nil {
		t.Error("expected error for missing dir")
	}
}

func writeFile(path, body string) error {
	return os.WriteFile(path, []byte(body), 0o644)
}

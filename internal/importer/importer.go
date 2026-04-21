package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/cards"
)

type Result struct {
	Inserted   int
	Updated    int
	Tombstoned int
	MediaCount int
	ChunkCount int
	Warnings   []string
}

// Import ingests a MyMind export folder into db, returning a summary. Cards
// present in db but absent from the new export get tombstoned. Tombstones
// older than 30 days are hard-deleted at the end of each import.
func Import(db *sql.DB, exportDir string) (Result, error) {
	var r Result
	if _, err := os.Stat(exportDir); err != nil {
		return r, fmt.Errorf("read export dir: %w", err)
	}

	start := time.Now().UTC()
	syncRes, err := db.Exec(`INSERT INTO sync_log(started_at, status) VALUES (?, 'running')`, start.Format(time.RFC3339))
	if err != nil {
		return r, err
	}
	syncID, _ := syncRes.LastInsertId()

	parsed, parseWarns, err := ParseCardsCSV(filepath.Join(exportDir, "cards.csv"))
	if err != nil {
		markSyncFailed(db, syncID, err)
		return r, err
	}
	r.Warnings = append(r.Warnings, parseWarns...)

	existing := map[string]bool{}
	rows, _ := db.Query(`SELECT mymind_id FROM cards WHERE deleted_at IS NULL`)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			existing[id] = true
		}
	}
	rows.Close()

	tx, err := db.Begin()
	if err != nil {
		markSyncFailed(db, syncID, err)
		return r, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	for _, c := range parsed {
		if err := upsertCard(tx, c, now); err != nil {
			tx.Rollback()
			markSyncFailed(db, syncID, err)
			return r, err
		}
		if existing[c.MyMindID] {
			r.Updated++
			delete(existing, c.MyMindID)
		} else {
			r.Inserted++
		}
		for _, ch := range Chunk(c.Body) {
			if _, err := tx.Exec(`INSERT INTO chunks(card_id, modality, text, start_offset, end_offset, checksum) VALUES (?, 'text', ?, ?, ?, ?)`,
				c.ID, ch.Text, ch.StartOffset, ch.EndOffset, ch.Checksum); err != nil {
				tx.Rollback()
				markSyncFailed(db, syncID, err)
				return r, err
			}
			r.ChunkCount++
		}
	}

	// Tombstone everything left in existing.
	for id := range existing {
		if _, err := tx.Exec(`UPDATE cards SET deleted_at = ? WHERE mymind_id = ?`, now, id); err != nil {
			tx.Rollback()
			markSyncFailed(db, syncID, err)
			return r, err
		}
		r.Tombstoned++
	}

	// Media scan inside the same transaction.
	// Honor the old media/ subfolder layout if present; otherwise walk the root.
	mediaRoot := filepath.Join(exportDir, "media")
	if _, err := os.Stat(mediaRoot); os.IsNotExist(err) {
		mediaRoot = exportDir
	}
	items, scanErr := ScanMedia(mediaRoot)
	if scanErr != nil {
		r.Warnings = append(r.Warnings, fmt.Sprintf("media scan: %v", scanErr))
	}
	for _, it := range items {
		if filepath.Base(it.Path) == "cards.csv" {
			continue
		}
		mediaKind := "other"
		switch {
		case strings.HasPrefix(it.Mime, "image/"):
			mediaKind = "image"
		case strings.HasPrefix(it.Mime, "video/"):
			mediaKind = "video"
		case it.Mime == "application/pdf":
			mediaKind = "document"
		}
		if _, err := tx.Exec(`INSERT INTO media(card_id, kind, path, sha256, mime) VALUES ('', ?, ?, ?, ?)`,
			mediaKind, it.Path, it.SHA256, it.Mime); err != nil {
			r.Warnings = append(r.Warnings, fmt.Sprintf("media insert %s: %v", it.Path, err))
			continue
		}
		r.MediaCount++
	}

	if err := tx.Commit(); err != nil {
		markSyncFailed(db, syncID, err)
		return r, err
	}

	// Hard-delete tombstones older than 30 days, outside the main transaction.
	cutoff := time.Now().UTC().Add(-30 * 24 * time.Hour).Format(time.RFC3339)
	if _, err := db.Exec(`DELETE FROM cards WHERE deleted_at IS NOT NULL AND deleted_at < ?`, cutoff); err != nil {
		r.Warnings = append(r.Warnings, fmt.Sprintf("hard-delete: %v", err))
	}

	finish := time.Now().UTC()
	_, _ = db.Exec(`UPDATE sync_log SET finished_at = ?, delta_count = ?, status = 'ok' WHERE id = ?`,
		finish.Format(time.RFC3339), r.Inserted+r.Updated+r.Tombstoned, syncID)

	return r, nil
}

// Note: the media `card_id` column has a foreign key to cards(id), and the
// empty-string insert above would normally violate it. SQLite's foreign-key
// enforcement is off by default, which lets the Phase 1 import land media rows
// without card mapping. Phase 2 will do real card-to-media joining when the
// export format exposes the linkage. For now this is intentional and noted.

func upsertCard(tx *sql.Tx, c cards.Card, updatedAt string) error {
	if _, err := tx.Exec(`INSERT INTO cards(id, mymind_id, kind, title, url, body, excerpt, source, captured_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(mymind_id) DO UPDATE SET
			title = excluded.title,
			url = excluded.url,
			body = excluded.body,
			excerpt = excluded.excerpt,
			source = excluded.source,
			captured_at = excluded.captured_at,
			updated_at = excluded.updated_at,
			deleted_at = NULL`,
		c.ID, c.MyMindID, string(c.Kind), c.Title, nullish(c.URL), nullish(c.Body), nullish(c.Excerpt), nullish(c.Source),
		c.CapturedAt.Format(time.RFC3339), updatedAt); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM tags WHERE card_id = ?`, c.ID); err != nil {
		return err
	}
	for _, t := range c.Tags {
		if _, err := tx.Exec(`INSERT INTO tags(card_id, tag) VALUES (?, ?)`, c.ID, t); err != nil {
			return err
		}
	}

	// Collect the current tags to rebuild tags_flat in FTS5.
	tagsFlat := ""
	trows, _ := tx.Query(`SELECT tag FROM tags WHERE card_id = ?`, c.ID)
	for trows.Next() {
		var t string
		if err := trows.Scan(&t); err == nil {
			if tagsFlat == "" {
				tagsFlat = t
			} else {
				tagsFlat += " " + t
			}
		}
	}
	trows.Close()

	// cards_fts is a contentless FTS5 table; UPDATE is not supported on
	// contentless tables. Use the FTS5 delete-then-reinsert pattern instead.
	// The cards_ai/cards_au trigger already seeded a row with empty tags_flat;
	// delete it and reinsert with the populated tags string.
	var rowid int64
	if err := tx.QueryRow(`SELECT rowid FROM cards WHERE id = ?`, c.ID).Scan(&rowid); err != nil {
		return err
	}
	var title, body string
	if err := tx.QueryRow(`SELECT title, coalesce(body,'') FROM cards WHERE id = ?`, c.ID).Scan(&title, &body); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO cards_fts(cards_fts, rowid, title, body, tags_flat) VALUES ('delete', ?, ?, ?, '')`,
		rowid, title, body); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO cards_fts(rowid, title, body, tags_flat) VALUES (?, ?, ?, ?)`,
		rowid, title, body, tagsFlat); err != nil {
		return err
	}

	// Replace chunks; caller re-inserts new chunks after this function returns.
	if _, err := tx.Exec(`DELETE FROM chunks WHERE card_id = ?`, c.ID); err != nil {
		return err
	}
	return nil
}

func nullish(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func markSyncFailed(db *sql.DB, syncID int64, err error) {
	if syncID == 0 {
		return
	}
	_, _ = db.Exec(`UPDATE sync_log SET finished_at = ?, status = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), "error: "+err.Error(), syncID)
}

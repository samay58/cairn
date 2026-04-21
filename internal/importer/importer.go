package importer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
	SkippedRows int
	Warnings   []string
}

type cardSnapshot struct {
	Kind       cards.Kind
	Title      string
	URL        string
	Body       string
	Excerpt    string
	Source     string
	CapturedAt string
	Tags       []string
	Deleted    bool
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
	if _, err := db.Exec(`UPDATE sync_log
		SET finished_at = ?, status = 'interrupted'
		WHERE finished_at IS NULL AND status = 'running'`, start.Format(time.RFC3339)); err != nil {
		return r, err
	}
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
	r.SkippedRows = len(parseWarns)

	snapshots, active, err := loadSnapshots(db)
	if err != nil {
		markSyncFailed(db, syncID, err)
		return r, err
	}
	seen := make(map[string]struct{}, len(parsed))
	for _, c := range parsed {
		if _, ok := seen[c.MyMindID]; ok {
			err := fmt.Errorf("duplicate mymind id %q in cards.csv", c.MyMindID)
			markSyncFailed(db, syncID, err)
			return r, err
		}
		seen[c.MyMindID] = struct{}{}
	}

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
		if snap, ok := snapshots[c.MyMindID]; ok {
			if snapshotChanged(snap, c) {
				r.Updated++
			}
		} else {
			r.Inserted++
		}
		delete(active, c.MyMindID)
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

	// Tombstone everything left in active.
	for id := range active {
		if _, err := tx.Exec(`UPDATE cards SET deleted_at = ? WHERE mymind_id = ?`, now, id); err != nil {
			tx.Rollback()
			markSyncFailed(db, syncID, err)
			return r, err
		}
		r.Tombstoned++
	}

	if _, err := tx.Exec(`DELETE FROM media`); err != nil {
		tx.Rollback()
		markSyncFailed(db, syncID, err)
		return r, err
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

func loadSnapshots(db *sql.DB) (map[string]cardSnapshot, map[string]bool, error) {
	rows, err := db.Query(`SELECT mymind_id, kind, title, coalesce(url, ''), coalesce(body, ''),
		coalesce(excerpt, ''), coalesce(source, ''), captured_at, deleted_at IS NOT NULL
		FROM cards`)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	snapshots := map[string]cardSnapshot{}
	active := map[string]bool{}
	for rows.Next() {
		var myMindID string
		var snap cardSnapshot
		var deleted int
		if err := rows.Scan(&myMindID, &snap.Kind, &snap.Title, &snap.URL, &snap.Body,
			&snap.Excerpt, &snap.Source, &snap.CapturedAt, &deleted); err != nil {
			return nil, nil, err
		}
		snap.Deleted = deleted != 0
		snapshots[myMindID] = snap
		if !snap.Deleted {
			active[myMindID] = true
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	tagRows, err := db.Query(`SELECT cards.mymind_id, tags.tag
		FROM cards JOIN tags ON tags.card_id = cards.id
		ORDER BY cards.mymind_id, tags.tag`)
	if err != nil {
		return nil, nil, err
	}
	defer tagRows.Close()

	for tagRows.Next() {
		var myMindID, tag string
		if err := tagRows.Scan(&myMindID, &tag); err != nil {
			return nil, nil, err
		}
		snap := snapshots[myMindID]
		snap.Tags = append(snap.Tags, tag)
		snapshots[myMindID] = snap
	}
	if err := tagRows.Err(); err != nil {
		return nil, nil, err
	}
	return snapshots, active, nil
}

func snapshotChanged(snap cardSnapshot, c cards.Card) bool {
	if snap.Deleted {
		return true
	}
	if snap.Kind != c.Kind ||
		snap.Title != c.Title ||
		snap.URL != c.URL ||
		snap.Body != c.Body ||
		snap.Excerpt != c.Excerpt ||
		snap.Source != c.Source ||
		snap.CapturedAt != c.CapturedAt.Format(time.RFC3339) {
		return true
	}
	return !equalTags(snap.Tags, c.Tags)
}

func equalTags(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for i := range leftCopy {
		if leftCopy[i] != rightCopy[i] {
			return false
		}
	}
	return true
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
				kind = excluded.kind,
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

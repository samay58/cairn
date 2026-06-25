package commands

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/samay58/cairn/internal/render"
)

type syncState struct {
	HasDB        bool
	LatestStatus string
	LatestFinish string
	LastGood     string
}

type importSummary struct {
	SourcePath  string `json:"source_path"`
	RowsRead    int    `json:"rows_read"`
	ValidCards  int    `json:"valid_cards"`
	Inserted    int    `json:"inserted"`
	Updated     int    `json:"updated"`
	Unchanged   int    `json:"unchanged"`
	Tombstoned  int    `json:"tombstoned"`
	SkippedRows int    `json:"skipped_rows"`
	Warnings    int    `json:"warnings"`
	Media       int    `json:"media"`
	Chunks      int    `json:"chunks"`
}

type exportSummary struct {
	TargetPath     string `json:"target_path"`
	FinishedAt     string `json:"finished_at"`
	CardsWritten   int    `json:"cards_written"`
	CardsUnchanged int    `json:"cards_unchanged"`
	MediaWritten   int    `json:"media_written"`
	MediaSkipped   int    `json:"media_skipped"`
	Warnings       int    `json:"warnings"`
	Status         string `json:"status"`
	Detail         string `json:"detail"`
}

func readSyncState(dbPath string) syncState {
	if _, err := os.Stat(dbPath); err != nil {
		return syncState{}
	}

	state := syncState{HasDB: true}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return state
	}
	defer db.Close()

	_ = db.QueryRow(`SELECT coalesce(status, ''), coalesce(finished_at, '')
		FROM sync_log ORDER BY id DESC LIMIT 1`).Scan(&state.LatestStatus, &state.LatestFinish)
	_ = db.QueryRow(`SELECT coalesce(finished_at, '')
		FROM sync_log WHERE status = 'ok' ORDER BY finished_at DESC LIMIT 1`).Scan(&state.LastGood)
	return state
}

func readLastImportSummary(dbPath string) importSummary {
	var out importSummary
	if _, err := os.Stat(dbPath); err != nil {
		return out
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return out
	}
	defer db.Close()
	_ = db.QueryRow(`SELECT
		coalesce(source_path, ''),
		rows_read,
		valid_cards,
		inserted_count,
		updated_count,
		unchanged_count,
		tombstoned_count,
		skipped_rows,
		warning_count,
		media_count,
		chunk_count
		FROM sync_log WHERE status='ok' ORDER BY finished_at DESC LIMIT 1`).Scan(
		&out.SourcePath,
		&out.RowsRead,
		&out.ValidCards,
		&out.Inserted,
		&out.Updated,
		&out.Unchanged,
		&out.Tombstoned,
		&out.SkippedRows,
		&out.Warnings,
		&out.Media,
		&out.Chunks,
	)
	return out
}

func readLastExportSummary(dbPath string) exportSummary {
	var out exportSummary
	if _, err := os.Stat(dbPath); err != nil {
		return out
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return out
	}
	defer db.Close()
	_ = db.QueryRow(`SELECT
		target_path,
		coalesce(finished_at, ''),
		cards_written,
		cards_unchanged,
		media_written,
		media_skipped,
		warning_count,
		status,
		coalesce(detail, '')
		FROM export_log ORDER BY id DESC LIMIT 1`).Scan(
		&out.TargetPath,
		&out.FinishedAt,
		&out.CardsWritten,
		&out.CardsUnchanged,
		&out.MediaWritten,
		&out.MediaSkipped,
		&out.Warnings,
		&out.Status,
		&out.Detail,
	)
	return out
}

func missingMediaSourceCount(dbPath string) int {
	if _, err := os.Stat(dbPath); err != nil {
		return 0
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return 0
	}
	defer db.Close()
	rows, err := db.Query(`SELECT path FROM media`)
	if err != nil {
		return 0
	}
	defer rows.Close()
	missing := 0
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			continue
		}
		if path == "" {
			missing++
			continue
		}
		if _, err := os.Stat(path); err != nil {
			missing++
		}
	}
	return missing
}

func formatImportError(op string, err error, state syncState) error {
	if err == nil {
		return nil
	}

	lines := []string{"import failed: " + op}
	for _, line := range render.WrapLines("detail: ", err.Error(), render.DefaultWidth) {
		lines = append(lines, line)
	}
	lines = append(lines, "last good: "+lastGoodLabel(state))
	return errors.New(strings.Join(lines, "\n"))
}

func lastGoodLabel(state syncState) string {
	if state.LastGood != "" {
		return state.LastGood
	}
	return "no successful import yet"
}

func syncStateLabel(state syncState) string {
	switch {
	case !state.HasDB:
		return "not started"
	case state.LatestStatus == "":
		return "database created, import not started"
	case state.LatestStatus == "ok":
		return "ok"
	case state.LatestStatus == "running":
		return "running"
	case state.LatestStatus == "interrupted":
		return "interrupted"
	case strings.HasPrefix(state.LatestStatus, "error:"):
		return "failed"
	default:
		return state.LatestStatus
	}
}

func latestFailureDetail(state syncState) string {
	if !strings.HasPrefix(state.LatestStatus, "error:") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(state.LatestStatus, "error:"))
}

func formatSyncLine(state syncState) string {
	label := syncStateLabel(state)
	if label == "ok" {
		return ""
	}
	if !state.HasDB {
		return ""
	}
	line := "sync      last attempt " + label
	if state.LastGood != "" {
		line += " · last good " + state.LastGood
	} else {
		line += " · no successful import yet"
	}
	return line
}

func formatSyncErrorLine(state syncState) string {
	detail := latestFailureDetail(state)
	if detail == "" {
		return ""
	}
	return fmt.Sprintf("detail    %s", detail)
}

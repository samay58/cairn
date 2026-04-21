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

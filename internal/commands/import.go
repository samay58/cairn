package commands

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/samay58/cairn/internal/importer"
	"github.com/samay58/cairn/internal/storage/sqlite"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			errOut := cmd.ErrOrStderr()
			exportDir := args[0]
			dbPath := cairnDBPath()
			state := readSyncState(dbPath)

			fmt.Fprintf(out, "Reading export from %s\n", exportDir)
			result, err := runImport(dbPath, exportDir, state)
			if err != nil {
				return err
			}
			fmt.Fprintf(out, "Rows: %d read, %d valid, %d skipped.\n",
				result.RowsRead, result.ValidCards, result.SkippedRows)
			fmt.Fprintf(out, "Cards: %d inserted, %d updated, %d unchanged, %d tombstoned.\n",
				result.Inserted, result.Updated, result.Unchanged, result.Tombstoned)
			fmt.Fprintf(out, "Media: %d files. Chunks: %d.\n", result.MediaCount, result.ChunkCount)
			switch {
			case result.SkippedRows > 0:
				fmt.Fprintf(out, "Warnings: %d rows skipped; details on stderr.\n", result.SkippedRows)
			case len(result.Warnings) > 0:
				fmt.Fprintf(out, "Warnings: %d issues; details on stderr.\n", len(result.Warnings))
			}
			fmt.Fprintln(out)
			if err := writeDatabaseLocation(out, dbPath); err != nil {
				return err
			}
			fmt.Fprintln(out, "Run `cairn search \"<query>\"` or `cairn find`.")
			for _, w := range result.Warnings {
				fmt.Fprintf(errOut, "warning: %s\n", w)
			}
			return nil
		},
	}
}

// runImport opens (and migrates) the database at dbPath and ingests exportDir.
// Errors come back already wrapped with recovery guidance.
func runImport(dbPath, exportDir string, state syncState) (importer.Result, error) {
	if _, err := os.Stat(exportDir); err != nil {
		return importer.Result{}, formatImportError("read export directory", err, state)
	}
	return withDB(dbPath, state, func(db *sql.DB) (importer.Result, error) {
		return importer.Import(db, exportDir)
	})
}

// withDB opens and migrates the database, runs ingest, and wraps any failure
// with the recovery guidance import errors carry.
func withDB(dbPath string, state syncState, ingest func(*sql.DB) (importer.Result, error)) (importer.Result, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return importer.Result{}, formatImportError("create cairn home", err, state)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return importer.Result{}, formatImportError("open database", err, state)
	}
	defer db.Close()
	if err := sqlite.Migrate(db); err != nil {
		return importer.Result{}, formatImportError("migrate database", err, state)
	}
	result, err := ingest(db)
	if err != nil {
		return result, formatImportError("ingest library", err, readSyncState(dbPath))
	}
	return result, nil
}

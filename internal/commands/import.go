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

			if _, err := os.Stat(exportDir); err != nil {
				return formatImportError("read export directory", err, state)
			}

			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return formatImportError("create cairn home", err, state)
			}
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return formatImportError("open database", err, state)
			}
			defer db.Close()
			if err := sqlite.Migrate(db); err != nil {
				return formatImportError("migrate database", err, state)
			}

			fmt.Fprintf(out, "Reading export from %s\n", exportDir)
			result, err := importer.Import(db, exportDir)
			if err != nil {
				return formatImportError("ingest export", err, readSyncState(dbPath))
			}
			fmt.Fprintf(out, "Parsed %d cards; %d inserted, %d updated, %d tombstoned.\n",
				result.Inserted+result.Updated, result.Inserted, result.Updated, result.Tombstoned)
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

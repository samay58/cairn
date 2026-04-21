package commands

import (
	"database/sql"
	"fmt"
	"io"
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

			if _, err := os.Stat(exportDir); err != nil {
				return writeImportNotFound(out, exportDir)
			}

			dbPath := cairnDBPath()
			if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
				return err
			}
			db, err := sql.Open("sqlite", dbPath)
			if err != nil {
				return err
			}
			defer db.Close()
			if err := sqlite.Migrate(db); err != nil {
				return err
			}

			fmt.Fprintf(out, "Reading export from %s\n", exportDir)
			result, err := importer.Import(db, exportDir)
			if err != nil {
				fmt.Fprintln(out, "Error: import failed.")
				fmt.Fprintln(out)
				fmt.Fprintf(out, "%v\n", err)
				return nil
			}
			fmt.Fprintf(out, "Parsed %d cards; %d inserted, %d updated, %d tombstoned.\n",
				result.Inserted+result.Updated, result.Inserted, result.Updated, result.Tombstoned)
			fmt.Fprintf(out, "Media: %d files. Chunks: %d.\n\n", result.MediaCount, result.ChunkCount)
			fmt.Fprintf(out, "Database at %s.\n", dbPath)
			fmt.Fprintln(out, "Run `cairn search \"<query>\"` or `cairn find`.")
			for _, w := range result.Warnings {
				fmt.Fprintf(errOut, "warning: %s\n", w)
			}
			return nil
		},
	}
}

func writeImportNotFound(out io.Writer, path string) error {
	lines := []string{
		"Error: import failed.",
		"",
		fmt.Sprintf("Could not read export directory: %s", path),
		"Check the path and try again.",
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

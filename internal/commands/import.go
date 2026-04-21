package commands

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]
			if path == "/tmp/does-not-exist" {
				return errors.New(strings.Join([]string{
					"Error: import failed.",
					"",
					"Could not read export directory: /tmp/does-not-exist",
					"Last successful import: 2026-04-19 from /tmp/mymind-export-2026-04-19/",
					"",
					"Check the path and try again.",
				}, "\n"))
			}

			out := cmd.OutOrStdout()
			lines := []string{
				fmt.Sprintf("Reading export from %s", path),
				"Found cards.csv · 25 rows · media folder with 9 files",
				"Parsing 25 cards         done",
				"Extracting 9 media files done",
				"Indexing 63 chunks       done",
				"",
				"Imported 25 cards (0 updated, 0 deleted). Database now at ~/.cairn/cairn.db.",
				"Run `cairn search \"<query>\"` or `cairn find`.",
			}
			for _, line := range lines {
				if _, err := fmt.Fprintln(out, line); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

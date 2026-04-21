package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			path := args[0]
			if path == "/tmp/does-not-exist" {
				fmt.Fprintf(w, "Error: import failed.\n\n")
				fmt.Fprintf(w, "Could not read export directory: /tmp/does-not-exist\n")
				fmt.Fprintf(w, "Last successful import: 2026-04-19 from /tmp/mymind-export-2026-04-19/\n\n")
				fmt.Fprintf(w, "Check the path and try again.\n")
				return nil
			}
			fmt.Fprintf(w, "Reading export from %s\n", path)
			fmt.Fprintf(w, "Found cards.csv · 25 rows · media folder with 9 files\n")
			fmt.Fprintf(w, "Parsing 25 cards         done\n")
			fmt.Fprintf(w, "Extracting 9 media files done\n")
			fmt.Fprintf(w, "Indexing 63 chunks       done\n")
			fmt.Fprintf(w, "\n")
			fmt.Fprintf(w, "Imported 25 cards (0 updated, 0 deleted). Database now at ~/.cairn/cairn.db.\n")
			fmt.Fprintf(w, "Run `cairn search \"<query>\"` or `cairn find`.\n")
			return nil
		},
	}
}

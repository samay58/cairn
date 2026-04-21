package commands

import (
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "cairn",
		Short: "Terminal-native bridge between MyMind and the tools you already use",
		Long:  "Cairn makes your MyMind library queryable from the terminal and a first-class context source for AI tools.\n\nPhase 0 is a design prototype with hand-authored output. Real storage and search arrive in Phase 1.",
	}

	root.PersistentFlags().Bool("json", false, "emit JSON output")
	root.PersistentFlags().Bool("jsonl", false, "emit JSONL output")
	root.PersistentFlags().Bool("plain", false, "emit plain text (default in Phase 0)")
	root.PersistentFlags().Int("limit", 0, "cap number of results (0 = default)")
	root.PersistentFlags().Bool("no-color", false, "disable color output (no-op in Phase 0)")

	root.AddCommand(
		newImportCmd(),
		newStatusCmd(),
		newSearchCmd(),
		newFindCmd(),
		newGetCmd(),
		newOpenCmd(),
		newPackCmd(),
		newAskCmd(),
		newExportCmd(),
		newConfigCmd(),
		newMCPCmd(),
	)
	return root
}

package commands

import "github.com/spf13/cobra"

func newFindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find",
		Short: "Full-screen fuzzy TUI (Phase 0: static mock)",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

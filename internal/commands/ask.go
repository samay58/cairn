package commands

import "github.com/spf13/cobra"

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask <question>",
		Short: "RAG synthesis with citations (Phase 4)",
		Args:  cobra.MinimumNArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

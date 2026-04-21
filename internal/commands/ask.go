package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask <question>",
		Short: "RAG synthesis with citations (Phase 4)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			lines := []string{
				"`cairn ask` ships in Phase 4, after the Phase 3 integrity gate passes.",
				"",
				"For now, use `cairn pack \"<question>\"` to hand a cited bundle.",
				"Give that bundle to your AI of choice.",
			}
			for _, line := range lines {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), line); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

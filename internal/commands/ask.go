package commands

import "github.com/spf13/cobra"

func newAskCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ask <question>",
		Short: "RAG synthesis with citations (Phase 4)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(
				"`cairn ask` ships in Phase 4, after the Phase 3 integrity gate passes.\n\n" +
					"For now, use `cairn pack \"<question>\"` to hand a cited bundle to your AI of choice.\n",
			))
			return err
		},
	}
}

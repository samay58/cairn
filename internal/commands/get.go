package commands

import "github.com/spf13/cobra"

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <card>",
		Short: "Render full card in terminal",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

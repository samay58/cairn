package commands

import "github.com/spf13/cobra"

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <card>",
		Short: "Open a card in the MyMind browser",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

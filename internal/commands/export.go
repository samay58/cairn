package commands

import "github.com/spf13/cobra"

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Mirror cards to Phoenix vault",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

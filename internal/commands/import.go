package commands

import "github.com/spf13/cobra"

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import <path>",
		Short: "Ingest a MyMind export folder",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

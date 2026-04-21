package commands

import "github.com/spf13/cobra"

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Edit config (Phase 0: shows defaults only)",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

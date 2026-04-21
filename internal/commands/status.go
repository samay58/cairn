package commands

import "github.com/spf13/cobra"

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Library size, last sync, MCP state, permissions",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

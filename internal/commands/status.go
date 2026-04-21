package commands

import "github.com/spf13/cobra"

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Library size, last sync, MCP state, permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(
				"cairn 0.0.0-phase0\n\n" +
					"library   25 cards · last import 2026-04-19 · 0 pending\n" +
					"storage   ~/.cairn/cairn.db (0 B) · media cache off\n" +
					"mcp       not installed\n" +
					"clients   none\n\n" +
					"Phase 0 prototype. Real storage lands in Phase 1.\n",
			))
			return err
		},
	}
}

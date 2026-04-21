package commands

import "github.com/spf13/cobra"

func newPackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack <query>",
		Short: "Cited context pack for an external AI",
		Args:  cobra.MinimumNArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
	cmd.Flags().String("for", "claude", "target profile: claude, chatgpt, cursor, markdown, json")
	return cmd
}

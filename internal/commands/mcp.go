package commands

import "github.com/spf13/cobra"

func newMCPCmd() *cobra.Command {
	mcp := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server, install, audit, permissions",
	}
	mcp.AddCommand(newMCPStartCmd(), newMCPInstallCmd(), newMCPAuditCmd(), newMCPPermissionsCmd())
	return mcp
}

func newMCPStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start",
		Short: "Start the local MCP server over stdio",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

func newMCPInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <client>",
		Short: "Write client config for cairn MCP",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

func newMCPAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Human-readable tool-call log",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

func newMCPPermissionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "permissions",
		Short: "Inspect and modify per-client scopes",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}
}

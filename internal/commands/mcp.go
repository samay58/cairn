package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

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
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			_, err := fmt.Fprint(out,
				"cairn mcp server (stdio). Phase 0 fake.\n"+
					"\n"+
					"Would expose 5 tools:\n"+
					"  search_mind(query, limit?, filters?)\n"+
					"  get_card(card_id, include_full_text?)\n"+
					"  get_related(card_id, limit?)\n"+
					"  create_context_pack(query, max_tokens?)\n"+
					"  save_to_mind(url_or_text, note?)              [disabled]\n"+
					"\n"+
					"Client config:\n"+
					"  claude-code     not installed\n"+
					"  claude-desktop  not installed\n"+
					"\n"+
					"Phase 0 exits immediately. Real server runs in Phase 3.\n",
			)
			return err
		},
	}
}

func newMCPInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install <client>",
		Short: "Write client config for cairn MCP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			switch args[0] {
			case "claude-code":
				_, err := fmt.Fprint(out,
					"Installing cairn MCP server for claude-code.\n"+
						"\n"+
						"Target config: ~/.claude/mcp.json\n"+
						"Existing servers detected: 2 (brave-search, filesystem)\n"+
						"\n"+
						"Proposed merge (new keys only):\n"+
						"  mcpServers.cairn = {\n"+
						"    \"command\": \"cairn\",\n"+
						"    \"args\": [\"mcp\", \"start\"]\n"+
						"  }\n"+
						"\n"+
						"Phase 0 would write this merge. Real install runs in Phase 3.\n"+
						"Run `cairn mcp permissions` to review tool-level scopes.\n",
				)
				return err
			case "claude-desktop":
				_, err := fmt.Fprint(out,
					"Installing cairn MCP server for claude-desktop.\n"+
						"\n"+
						"Target config: ~/Library/Application Support/Claude/claude_desktop_config.json\n"+
						"Existing servers detected: 4 (brave-search, filesystem, exa, perplexity-ask)\n"+
						"\n"+
						"Proposed merge (new keys only):\n"+
						"  mcpServers.cairn = {\n"+
						"    \"command\": \"cairn\",\n"+
						"    \"args\": [\"mcp\", \"start\"]\n"+
						"  }\n"+
						"\n"+
						"Phase 0 would write this merge. Real install runs in Phase 3.\n"+
						"Run `cairn mcp permissions` to review tool-level scopes.\n",
				)
				return err
			default:
				_, err := fmt.Fprintf(out, "Unknown client %q. Phase 0 supports: claude-code, claude-desktop.\n", args[0])
				return err
			}
		},
	}
}

func newMCPAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Human-readable tool-call log",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			_, err := fmt.Fprint(out,
				"Today\n"+
					"\n"+
					"  10:42  Claude Code searched \"OAuth MCP authorization\"\n"+
					"         Returned 5 snippets\n"+
					"\n"+
					"  10:43  Claude Code requested full content for @2\n"+
					"         Allowed once by user\n"+
					"\n"+
					"  10:49  Cursor searched \"SQLite FTS5\"\n"+
					"         Returned 8 snippets\n"+
					"\n"+
					"Yesterday\n"+
					"\n"+
					"  22:11  Claude Desktop searched \"prompt injection mitigations\"\n"+
					"         Returned 4 snippets\n"+
					"\n"+
					"Phase 0: sample audit rows. Real log reads from the mcp_audit table in Phase 3.\n",
			)
			return err
		},
	}
}

func newMCPPermissionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "permissions",
		Short: "Inspect and modify per-client scopes",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			_, err := fmt.Fprint(out,
				"Per-client MCP permissions\n"+
					"\n"+
					"  claude-code\n"+
					"    search_mind           allow\n"+
					"    get_related           allow\n"+
					"    get_card              prompt (include_full_text=true)\n"+
					"    create_context_pack   allow\n"+
					"    save_to_mind          deny\n"+
					"\n"+
					"  claude-desktop\n"+
					"    search_mind           allow\n"+
					"    get_related           allow\n"+
					"    get_card              prompt (include_full_text=true)\n"+
					"    create_context_pack   allow\n"+
					"    save_to_mind          deny\n"+
					"\n"+
					"Phase 0: values are the privacy-first defaults. Edit ~/.cairn/config.toml in Phase 3 to change them.\n",
			)
			return err
		},
	}
}

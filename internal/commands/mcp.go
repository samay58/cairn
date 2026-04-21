package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/samay58/cairn/internal/render"
	"github.com/spf13/cobra"
)

type mcpInstallPreview struct {
	Client   string   `json:"client"`
	Target   string   `json:"target"`
	Existing []string `json:"existing"`
	Command  []string `json:"command"`
	Phase    []string `json:"phase"`
}

type mcpAuditEntry struct {
	Period string `json:"period"`
	Time   string `json:"time"`
	Action string `json:"action"`
	Result string `json:"result"`
}

type mcpPermissionSet struct {
	Client string `json:"client"`
	Rules  []struct {
		Tool  string `json:"tool"`
		Value string `json:"value"`
	} `json:"rules"`
}

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
			lines := []string{
				"cairn mcp server (stdio). Phase 0 fake.",
				"",
				"Would expose 5 tools:",
				"  search_mind(query, limit?, filters?)",
				"  get_card(card_id, include_full_text?)",
				"  get_related(card_id, limit?)",
				"  create_context_pack(query, max_tokens?)",
				"  save_to_mind(url_or_text, note?)              [disabled]",
				"",
				"Client config:",
				"  claude-code     not installed",
				"  claude-desktop  not installed",
				"",
				"Phase 0 exits immediately. Real server runs in Phase 3.",
			}
			for _, line := range lines {
				if _, err := fmt.Fprintln(out, line); err != nil {
					return err
				}
			}
			return nil
		},
	}
}

func newMCPInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <client>",
		Short: "Write client config for cairn MCP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := args[0]
			switch client {
			case "manual":
				return writeManualInstall(cmd.OutOrStdout())
			default:
				preview, ok := mcpInstallPreviews()[client]
				if !ok {
					return fmt.Errorf("unknown client %q. Phase 0 supports: claude-code, claude-desktop, manual", client)
				}
				return writeInstallPreview(cmd.OutOrStdout(), preview)
			}
		},
	}
	return cmd
}

func newMCPAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Human-readable tool-call log",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}
			limit, err := limitValue(cmd)
			if err != nil {
				return err
			}

			entries := applyLimit(mcpAuditEntries(), limit)
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(entries))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL(entries))
			default:
				err = writeAuditPlain(out, entries)
			}
			return err
		},
	}
	addListFlags(cmd)
	return cmd
}

func newMCPPermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "Inspect and modify per-client scopes",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}

			permissions := mcpPermissions()
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(permissions))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL(permissions))
			default:
				err = writePermissionsPlain(out, permissions)
			}
			return err
		},
	}
	addOutputFlags(cmd)
	return cmd
}

func mcpInstallPreviews() map[string]mcpInstallPreview {
	return map[string]mcpInstallPreview{
		"claude-code": {
			Client:   "claude-code",
			Target:   "~/.claude/mcp.json",
			Existing: []string{"brave-search", "filesystem"},
			Command:  []string{"cairn", "mcp", "start"},
			Phase: []string{
				"Phase 0 would write this merge. Real install runs in Phase 3.",
				"Run `cairn mcp permissions` to review tool-level scopes.",
			},
		},
		"claude-desktop": {
			Client:   "claude-desktop",
			Target:   "~/Library/Application Support/Claude/claude_desktop_config.json",
			Existing: []string{"brave-search", "filesystem", "exa", "perplexity-ask"},
			Command:  []string{"cairn", "mcp", "start"},
			Phase: []string{
				"Phase 0 would write this merge. Real install runs in Phase 3.",
				"Run `cairn mcp permissions` to review tool-level scopes.",
			},
		},
	}
}

func writeInstallPreview(out io.Writer, preview mcpInstallPreview) error {
	lines := []string{
		fmt.Sprintf("Installing cairn MCP server for %s.", preview.Client),
		"",
		fmt.Sprintf("Target config: %s", preview.Target),
		fmt.Sprintf("Existing servers detected: %d (%s)", len(preview.Existing), strings.Join(preview.Existing, ", ")),
		"",
		"Proposed merge (new keys only):",
		"  mcpServers.cairn = {",
		"    \"command\": \"cairn\",",
		"    \"args\": [\"mcp\", \"start\"]",
		"  }",
		"",
		preview.Phase[0],
		preview.Phase[1],
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func writeManualInstall(out io.Writer) error {
	lines := []string{
		"Copy-pasteable cairn MCP server snippet for any MCP-compatible client:",
		"",
		"{",
		"  \"cairn\": {",
		"    \"command\": \"cairn\",",
		"    \"args\": [\"mcp\", \"start\"]",
		"  }",
		"}",
		"",
		"Merge this into your client's MCP config under its server key.",
		"Phase 0 emits this snippet only.",
		"`cairn mcp install claude-code` and `claude-desktop` write in Phase 3.",
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func mcpAuditEntries() []mcpAuditEntry {
	return []mcpAuditEntry{
		{Period: "Today", Time: "10:42", Action: `Claude Code searched "OAuth MCP authorization"`, Result: "Returned 5 snippets"},
		{Period: "Today", Time: "10:43", Action: "Claude Code requested full content for @2", Result: "Allowed once by user"},
		{Period: "Today", Time: "10:49", Action: `Cursor searched "SQLite FTS5"`, Result: "Returned 8 snippets"},
		{Period: "Yesterday", Time: "22:11", Action: `Claude Desktop searched "prompt injection mitigations"`, Result: "Returned 4 snippets"},
	}
}

func writeAuditPlain(out io.Writer, entries []mcpAuditEntry) error {
	currentPeriod := ""
	for _, entry := range entries {
		if entry.Period != currentPeriod {
			if _, err := fmt.Fprintln(out, entry.Period); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			currentPeriod = entry.Period
		}
		if _, err := fmt.Fprintf(out, "  %s  %s\n", entry.Time, entry.Action); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "         %s\n\n", entry.Result); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(out, "Phase 0: sample audit rows. Real log reads from the mcp_audit table in Phase 3.")
	return err
}

func mcpPermissions() []mcpPermissionSet {
	makeRules := func() []struct {
		Tool  string `json:"tool"`
		Value string `json:"value"`
	} {
		return []struct {
			Tool  string `json:"tool"`
			Value string `json:"value"`
		}{
			{Tool: "search_mind", Value: "allow"},
			{Tool: "get_related", Value: "allow"},
			{Tool: "get_card", Value: "prompt (include_full_text=true)"},
			{Tool: "create_context_pack", Value: "allow"},
			{Tool: "save_to_mind", Value: "deny"},
		}
	}
	return []mcpPermissionSet{
		{Client: "claude-code", Rules: makeRules()},
		{Client: "claude-desktop", Rules: makeRules()},
	}
}

func writePermissionsPlain(out io.Writer, permissions []mcpPermissionSet) error {
	if _, err := fmt.Fprintln(out, "Per-client MCP permissions"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	for i, permission := range permissions {
		if _, err := fmt.Fprintf(out, "  %s\n", permission.Client); err != nil {
			return err
		}
		for _, rule := range permission.Rules {
			if _, err := fmt.Fprintf(out, "    %-21s %s\n", rule.Tool, rule.Value); err != nil {
				return err
			}
		}
		if i < len(permissions)-1 {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "Phase 0 values are the privacy-first defaults."); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, "Edit ~/.cairn/config.toml in Phase 3 to change them.")
	return err
}

package commands

import (
	"fmt"
	"io"

	"github.com/samay58/cairn/internal/render"
	"github.com/spf13/cobra"
)

type configView struct {
	Path    string `json:"path"`
	Storage struct {
		CacheFullContent bool `json:"cache_full_content"`
		CacheMedia       bool `json:"cache_media"`
	} `json:"storage"`
	Embeddings struct {
		Model  string `json:"model"`
		Device string `json:"device"`
	} `json:"embeddings"`
	LLM struct {
		Provider string `json:"provider"`
		Model    string `json:"model"`
	} `json:"llm"`
	MCP struct {
		DefaultPermissions struct {
			SearchMind bool   `json:"search_mind"`
			GetRelated bool   `json:"get_related"`
			GetCard    string `json:"get_card"`
			SaveToMind bool   `json:"save_to_mind"`
		} `json:"default_permissions"`
	} `json:"mcp"`
	Phase []string `json:"phase"`
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Edit config (Phase 0: shows defaults only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}

			config := buildConfigView()
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(config))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL([]configView{config}))
			default:
				err = writeConfigPlain(out, config)
			}
			return err
		},
	}
	addOutputFlags(cmd)
	return cmd
}

func buildConfigView() configView {
	config := configView{
		Path: "~/.cairn/config.toml",
		Phase: []string{
			"Phase 0 values are hand-authored defaults.",
			"Real config reads and writes arrive in Phase 1.",
		},
	}
	config.Storage.CacheFullContent = false
	config.Storage.CacheMedia = false
	config.Embeddings.Model = "minilm-l6-v2"
	config.Embeddings.Device = "cpu"
	config.LLM.Provider = "anthropic"
	config.LLM.Model = "claude-opus-4-7"
	config.MCP.DefaultPermissions.SearchMind = true
	config.MCP.DefaultPermissions.GetRelated = true
	config.MCP.DefaultPermissions.GetCard = "prompt"
	config.MCP.DefaultPermissions.SaveToMind = false
	return config
}

func writeConfigPlain(out io.Writer, config configView) error {
	lines := []string{
		config.Path + " (Phase 0 defaults)",
		"",
		"[storage]",
		"cache_full_content = false",
		"cache_media = false",
		"",
		"[embeddings]",
		"model = \"minilm-l6-v2\"",
		"device = \"cpu\"",
		"",
		"[llm]",
		"provider = \"anthropic\"",
		"model = \"claude-opus-4-7\"",
		"",
		"[mcp.default_permissions]",
		"search_mind = true",
		"get_related = true",
		"get_card = \"prompt\"",
		"save_to_mind = false",
		"",
		config.Phase[0],
		config.Phase[1],
	}
	for _, line := range lines {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

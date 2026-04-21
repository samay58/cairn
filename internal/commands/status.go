package commands

import (
	"fmt"
	"io"

	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

type statusView struct {
	Version string `json:"version"`
	Library struct {
		Cards      int    `json:"cards"`
		LastImport string `json:"last_import"`
		Pending    int    `json:"pending"`
	} `json:"library"`
	Storage struct {
		Path       string `json:"path"`
		Size       string `json:"size"`
		MediaCache string `json:"media_cache"`
	} `json:"storage"`
	MCP struct {
		State   string   `json:"state"`
		Clients []string `json:"clients"`
	} `json:"mcp"`
	Permissions string `json:"permissions"`
	Phase       string `json:"phase"`
}

func newStatusCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Library size, last sync, MCP state, permissions",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}

			status := buildStatusView(src)
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(status))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL([]statusView{status}))
			default:
				err = writeStatusPlain(out, status)
			}
			return err
		},
	}
	addOutputFlags(cmd)
	return cmd
}

func buildStatusView(src source.Source) statusView {
	status := statusView{
		Version:     "cairn 0.0.0-phase0",
		Permissions: "Default search and related allow. Full content prompts.",
		Phase:       "Phase 0 prototype. Real storage lands in Phase 1.",
	}
	status.Library.Cards = src.Count()
	status.Library.LastImport = "2026-04-19"
	status.Library.Pending = 0
	status.Storage.Path = "~/.cairn/cairn.db"
	status.Storage.Size = "0 B"
	status.Storage.MediaCache = "off"
	status.MCP.State = "not installed"
	status.MCP.Clients = []string{}
	return status
}

func writeStatusPlain(out io.Writer, status statusView) error {
	_, err := fmt.Fprintf(out,
		"%s\n\n"+
			"library      %d cards · last import %s · %d pending\n"+
			"storage      %s (%s) · media cache %s\n"+
			"mcp          %s\n"+
			"clients      none\n"+
			"permissions  default search and related allow · full content prompts\n\n"+
			"%s\n",
		status.Version,
		status.Library.Cards,
		status.Library.LastImport,
		status.Library.Pending,
		status.Storage.Path,
		status.Storage.Size,
		status.Storage.MediaCache,
		status.MCP.State,
		status.Phase,
	)
	return err
}

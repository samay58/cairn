package commands

import (
	"fmt"
	"io"
	"os"

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
		Version:     "cairn 0.1.0-phase1",
		Permissions: "Default search and related allow. Full content prompts.",
		Phase:       "Phase 1. Import-backed search. Other commands ship later.",
	}
	status.Library.Cards = src.Count()
	status.MCP.State = "not installed"
	status.MCP.Clients = []string{}

	if ts, ok := src.LastImport(); ok {
		status.Library.LastImport = ts.Format("2006-01-02T15:04:05Z")
		status.Storage.Path = cairnDBPath()
		status.Storage.Size = dbSizeHuman(status.Storage.Path)
		status.Storage.MediaCache = "off"
	} else {
		status.Library.LastImport = "none"
		status.Storage.Path = "run `cairn import <path>` to create a database"
		status.Storage.Size = "0 B"
		status.Storage.MediaCache = "off"
	}
	status.Library.Pending = 0
	return status
}

func writeStatusPlain(out io.Writer, s statusView) error {
	if _, err := fmt.Fprintf(out, "%s\n\n", s.Version); err != nil {
		return err
	}
	if s.Library.LastImport == "none" {
		if _, err := fmt.Fprintf(out, "library   %d cards (fixtures; no database yet)\n", s.Library.Cards); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "storage   %s\n", s.Storage.Path); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(out, "library   %d cards · last import %s · %d pending\n", s.Library.Cards, s.Library.LastImport, s.Library.Pending); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "storage   %s (%s) · media cache %s\n", s.Storage.Path, s.Storage.Size, s.Storage.MediaCache); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(out, "mcp       %s\n", s.MCP.State); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "clients   none"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, s.Phase)
	return err
}

func dbSizeHuman(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}
	n := info.Size()
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/1024/1024)
	}
}

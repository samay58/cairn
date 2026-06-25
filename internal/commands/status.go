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
	Import  importSummary `json:"import"`
	Export  exportSummary `json:"export"`
	Storage struct {
		Path                string `json:"path"`
		Size                string `json:"size"`
		MediaCache          string `json:"media_cache"`
		MediaSourcesMissing int    `json:"media_sources_missing"`
	} `json:"storage"`
	MCP struct {
		State   string   `json:"state"`
		Clients []string `json:"clients"`
	} `json:"mcp"`
	Commands struct {
		Real           []string `json:"real"`
		NotImplemented []string `json:"not_implemented"`
		Designed       []string `json:"designed"`
	} `json:"commands"`
	Sync struct {
		State    string `json:"state"`
		LastGood string `json:"last_good"`
		Detail   string `json:"detail,omitempty"`
	} `json:"sync"`
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
	dbPath := cairnDBPath()
	sync := readSyncState(dbPath)
	status := statusView{
		Version:     "cairn 0.1.0-phase2a",
		Permissions: "Default search and related allow. Full content prompts.",
		Phase:       "Phase 2a. Import, search, get, open, and export are real.",
	}
	status.Library.Cards = src.Count()
	status.Import = readLastImportSummary(dbPath)
	status.Export = readLastExportSummary(dbPath)
	status.MCP.State = "not installed"
	status.MCP.Clients = []string{}
	status.Commands.Real = []string{"import", "status", "search", "get", "open", "export"}
	status.Commands.NotImplemented = []string{"find", "pack", "ask", "mcp"}
	status.Commands.Designed = []string{"config"}
	status.Sync.State = syncStateLabel(sync)
	status.Sync.LastGood = lastGoodLabel(sync)
	status.Sync.Detail = latestFailureDetail(sync)

	switch {
	case !sync.HasDB:
		status.Library.LastImport = "none"
		status.Storage.Path = "run `cairn import <path>` to create a database"
		status.Storage.Size = "0 B"
		status.Storage.MediaCache = "off"
	case hasSuccessfulImport(src):
		ts, _ := src.LastImport()
		status.Library.LastImport = ts.Format("2006-01-02T15:04:05Z")
		status.Storage.Path = dbPath
		status.Storage.Size = dbSizeHuman(status.Storage.Path)
		status.Storage.MediaCache = "off"
		status.Storage.MediaSourcesMissing = missingMediaSourceCount(dbPath)
	default:
		status.Library.LastImport = "none"
		status.Storage.Path = dbPath
		status.Storage.Size = dbSizeHuman(status.Storage.Path)
		status.Storage.MediaCache = "off"
		status.Storage.MediaSourcesMissing = missingMediaSourceCount(dbPath)
	}
	status.Library.Pending = 0
	return status
}

func writeStatusPlain(out io.Writer, s statusView) error {
	if _, err := fmt.Fprintf(out, "%s\n\n", s.Version); err != nil {
		return err
	}
	switch {
	case s.Storage.Path == "run `cairn import <path>` to create a database":
		if _, err := fmt.Fprintf(out, "library   %d cards (fixtures; no database yet)\n", s.Library.Cards); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "storage   %s\n", s.Storage.Path); err != nil {
			return err
		}
	case s.Library.LastImport != "none":
		if _, err := fmt.Fprintf(out, "library   %d cards · last import %s · %d pending\n", s.Library.Cards, s.Library.LastImport, s.Library.Pending); err != nil {
			return err
		}
		if err := writeStorageLocation(out, s.Storage.Path, s.Storage.Size, s.Storage.MediaCache); err != nil {
			return err
		}
	default:
		if _, err := fmt.Fprintf(out, "library   %d cards · import state %s · %d pending\n", s.Library.Cards, s.Sync.State, s.Library.Pending); err != nil {
			return err
		}
		if err := writeStorageLocation(out, s.Storage.Path, s.Storage.Size, s.Storage.MediaCache); err != nil {
			return err
		}
	}
	if s.Library.LastImport == "none" && s.Storage.Path != "run `cairn import <path>` to create a database" {
		line := "sync      last attempt " + s.Sync.State
		if s.Sync.LastGood != "" && s.Sync.LastGood != "no successful import yet" {
			line += " · last good " + s.Sync.LastGood
		} else {
			line += " · no successful import yet"
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	if s.Import.ValidCards > 0 {
		if _, err := fmt.Fprintf(out, "import    %d read · %d valid · %d skipped · %d warnings\n",
			s.Import.RowsRead, s.Import.ValidCards, s.Import.SkippedRows, s.Import.Warnings); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(out, "changes   %d inserted · %d updated · %d unchanged · %d tombstoned\n",
			s.Import.Inserted, s.Import.Updated, s.Import.Unchanged, s.Import.Tombstoned); err != nil {
			return err
		}
		if s.Storage.MediaSourcesMissing > 0 {
			if _, err := fmt.Fprintf(out, "media     %d source files missing\n", s.Storage.MediaSourcesMissing); err != nil {
				return err
			}
		}
	}
	if s.Export.TargetPath != "" {
		if _, err := fmt.Fprintf(out, "export    %s · %d cards · %d media · %d warnings\n",
			s.Export.Status,
			s.Export.CardsWritten+s.Export.CardsUnchanged,
			s.Export.MediaWritten+s.Export.MediaSkipped,
			s.Export.Warnings); err != nil {
			return err
		}
		for _, line := range wrapFilesystemPath("          ", s.Export.TargetPath, render.DefaultWidth) {
			if _, err := fmt.Fprintln(out, line); err != nil {
				return err
			}
		}
		if isDefaultPhoenixExportPath(s.Export.TargetPath) {
			if _, err := fmt.Fprintln(out, "qmd       cd ~/phoenix && qmd update"); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(out, "mcp       %s\n", s.MCP.State); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "clients   none"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "commands  real: import, status, search, get, open, export"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out, "          not implemented: find, pack, ask, mcp"); err != nil {
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

func hasSuccessfulImport(src source.Source) bool {
	_, ok := src.LastImport()
	return ok
}

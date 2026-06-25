package commands

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/samay58/cairn/internal/phoenix"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

type exportView struct {
	CardsWritten   int      `json:"cards_written"`
	CardsUnchanged int      `json:"cards_unchanged"`
	MediaWritten   int      `json:"media_written"`
	MediaSkipped   int      `json:"media_skipped"`
	Warnings       []string `json:"warnings"`
	Path           string   `json:"path"`
	DryRun         bool     `json:"dry_run"`
}

func newExportCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Mirror cards to the Phoenix vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}
			dry, _ := cmd.Flags().GetBool("dry-run")
			to, _ := cmd.Flags().GetString("to")
			if to == "" {
				to = defaultExportRoot()
			}

			// Refuse to write fixture data into a real Phoenix vault. Without
			// a real import there is nothing to mirror.
			if _, imported := src.LastImport(); !imported {
				_, err := fmt.Fprintln(cmd.OutOrStdout(),
					"No import recorded yet. Run `cairn import <path>` first.")
				return err
			}
			if importPath := lastImportSourcePath(cairnDBPath()); importPath != "" && samePathCaseInsensitive(to, importPath) {
				return fmt.Errorf("export target matches the last import path: %s", to)
			}

			bundles := collectBundles(src)
			w := &phoenix.Writer{Root: to, DryRun: dry}
			started := time.Now().UTC()
			rep, werr := w.Write(bundles)
			if werr != nil {
				if !dry {
					recordExportSummary(cairnDBPath(), started, to, rep, "error", werr.Error())
				}
				return fmt.Errorf("export to %s: %w", to, werr)
			}
			if !dry {
				recordExportSummary(cairnDBPath(), started, to, rep, "ok", "")
			}
			view := exportView{
				CardsWritten:   rep.CardsWritten,
				CardsUnchanged: rep.CardsUnchanged,
				MediaWritten:   rep.MediaWritten,
				MediaSkipped:   rep.MediaSkipped,
				Warnings:       rep.Warnings,
				Path:           to,
				DryRun:         dry,
			}
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(view))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL([]exportView{view}))
			default:
				err = writeExportPlain(out, view)
			}
			return err
		},
	}
	addOutputFlags(cmd)
	cmd.Flags().Bool("dry-run", false, "Preview without writing")
	cmd.Flags().String("to", "", "Vault root (defaults to ~/phoenix/04-knowledge-base/research-archive/mymind-cards/)")
	return cmd
}

func collectBundles(src source.Source) []phoenix.CardBundle {
	all := src.All()
	out := make([]phoenix.CardBundle, 0, len(all))
	for _, c := range all {
		out = append(out, phoenix.CardBundle{Card: c, Media: src.MediaFor(c.ID)})
	}
	return out
}

func defaultExportRoot() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "phoenix", "04-knowledge-base", "research-archive", "mymind-cards")
}

func writeExportPlain(out io.Writer, v exportView) error {
	verb := "Wrote"
	if v.DryRun {
		verb = "Would write"
	}
	if _, err := fmt.Fprintf(out, "%s %d cards to %s\n", verb, v.CardsWritten, v.Path); err != nil {
		return err
	}
	if v.CardsUnchanged > 0 {
		if _, err := fmt.Fprintf(out, "  %d cards unchanged\n", v.CardsUnchanged); err != nil {
			return err
		}
	}
	if v.MediaWritten > 0 || v.MediaSkipped > 0 {
		if _, err := fmt.Fprintf(out, "  media: %d written, %d skipped\n", v.MediaWritten, v.MediaSkipped); err != nil {
			return err
		}
	}
	if !v.DryRun {
		if _, err := fmt.Fprintf(out, "  mirror: %d cards, %d media files\n",
			v.CardsWritten+v.CardsUnchanged, v.MediaWritten+v.MediaSkipped); err != nil {
			return err
		}
	}
	for _, wn := range v.Warnings {
		if _, err := fmt.Fprintf(out, "  warning: %s\n", wn); err != nil {
			return err
		}
	}
	if v.DryRun {
		if _, err := fmt.Fprintln(out, "Remove --dry-run to write."); err != nil {
			return err
		}
	}
	if !v.DryRun && isDefaultPhoenixExportPath(v.Path) {
		if _, err := fmt.Fprintln(out, "Next: cd ~/phoenix && qmd update"); err != nil {
			return err
		}
	}
	return nil
}

func lastImportSourcePath(dbPath string) string {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return ""
	}
	defer db.Close()
	var path string
	_ = db.QueryRow(`SELECT coalesce(source_path, '')
		FROM sync_log WHERE status='ok' ORDER BY finished_at DESC LIMIT 1`).Scan(&path)
	return path
}

func samePathCaseInsensitive(left, right string) bool {
	leftAbs, leftErr := filepath.Abs(filepath.Clean(left))
	rightAbs, rightErr := filepath.Abs(filepath.Clean(right))
	if leftErr != nil || rightErr != nil {
		return strings.EqualFold(filepath.Clean(left), filepath.Clean(right))
	}
	return strings.EqualFold(leftAbs, rightAbs)
}

func recordExportSummary(dbPath string, started time.Time, target string, rep phoenix.WriteReport, status, detail string) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return
	}
	defer db.Close()
	_, _ = db.Exec(`INSERT INTO export_log(
		started_at, finished_at, target_path, cards_written, cards_unchanged,
		media_written, media_skipped, warning_count, status, detail
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		started.Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
		target,
		rep.CardsWritten,
		rep.CardsUnchanged,
		rep.MediaWritten,
		rep.MediaSkipped,
		len(rep.Warnings),
		status,
		detail)
}

func isDefaultPhoenixExportPath(path string) bool {
	return samePathCaseInsensitive(path, defaultExportRoot())
}

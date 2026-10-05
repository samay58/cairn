package commands

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/samay58/cairn/internal/phoenix"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/storage/sqlite"
	"github.com/spf13/cobra"
)

type syncImportView struct {
	Inserted   int `json:"inserted"`
	Updated    int `json:"updated"`
	Unchanged  int `json:"unchanged"`
	Tombstoned int `json:"tombstoned"`
	Skipped    int `json:"skipped_rows"`
	Media      int `json:"media"`
}

type syncView struct {
	Source   string               `json:"source"`
	Import   syncImportView       `json:"import"`
	Export   exportView           `json:"export"`
	Relink   phoenix.RelinkReport `json:"relink"`
	Warnings []string             `json:"warnings"`
}

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [export-dir]",
		Short: "Import a MyMind export, mirror it to the vault, and re-link raw media",
		Long: "sync runs import, export, and a media re-link in one step. The export folder\n" +
			"defaults to ~/phoenix/Clippings/mymind and the vault to\n" +
			"~/phoenix/04-knowledge-base/mymind-cards. Every step is idempotent.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}
			exportDir := defaultRawExportDir()
			if len(args) == 1 {
				exportDir = args[0]
			}
			to, _ := cmd.Flags().GetString("to")
			if to == "" {
				to = defaultExportRoot()
			}
			if samePathCaseInsensitive(to, exportDir) {
				return fmt.Errorf("vault path matches the export folder: %s", to)
			}

			dbPath := cairnDBPath()
			result, err := runImport(dbPath, exportDir, readSyncState(dbPath))
			if err != nil {
				return err
			}
			// Open the source after the import so the export sees the new rows.
			src, err := sqlite.Open(dbPath)
			if err != nil {
				return fmt.Errorf("open %s: %w", dbPath, err)
			}
			defer src.Close()
			exp, err := mirrorToVault(src, to, false)
			if err != nil {
				return err
			}
			rel, err := phoenix.RelinkRaw(exportDir, to)
			if err != nil {
				return fmt.Errorf("re-link media in %s: %w", exportDir, err)
			}

			view := syncView{
				Source: exportDir,
				Import: syncImportView{
					Inserted: result.Inserted, Updated: result.Updated,
					Unchanged: result.Unchanged, Tombstoned: result.Tombstoned,
					Skipped: result.SkippedRows, Media: result.MediaCount,
				},
				Export:   exp,
				Relink:   rel,
				Warnings: append(result.Warnings, exp.Warnings...),
			}
			for _, m := range rel.Missing {
				view.Warnings = append(view.Warnings, "no vault copy to link: "+m)
			}
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(view))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL([]syncView{view}))
			default:
				err = writeSyncPlain(out, view)
			}
			return err
		},
	}
	addOutputFlags(cmd)
	cmd.Flags().String("to", "", "Vault root (defaults to ~/phoenix/04-knowledge-base/mymind-cards/)")
	return cmd
}

func defaultRawExportDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "phoenix", "Clippings", "mymind")
}

func writeSyncPlain(out io.Writer, v syncView) error {
	lines := []string{
		fmt.Sprintf("Imported %s: %d new, %d updated, %d unchanged, %d tombstoned.",
			v.Source, v.Import.Inserted, v.Import.Updated, v.Import.Unchanged, v.Import.Tombstoned),
		fmt.Sprintf("Mirrored to %s: %d written, %d unchanged; media %d written, %d already there.",
			v.Export.Path, v.Export.CardsWritten, v.Export.CardsUnchanged, v.Export.MediaWritten, v.Export.MediaSkipped),
		fmt.Sprintf("Raw media: %d re-linked, %d already linked, %d without a vault copy.",
			v.Relink.Linked, v.Relink.AlreadyLinked, len(v.Relink.Missing)),
	}
	for _, w := range v.Warnings {
		lines = append(lines, "warning: "+w)
	}
	for _, l := range lines {
		if _, err := fmt.Fprintln(out, l); err != nil {
			return err
		}
	}
	return nil
}

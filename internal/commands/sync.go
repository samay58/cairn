package commands

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/samay58/cairn/internal/importer"
	"github.com/samay58/cairn/internal/mymindapi"
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

type syncAPIView struct {
	Objects            int `json:"objects"`
	Credits            int `json:"credits"`
	SustainedRemaining int `json:"sustained_remaining"`
	SustainedLimit     int `json:"sustained_limit"`
	AttachmentsFetched int `json:"attachments_fetched"`
}

type syncView struct {
	Source   string               `json:"source"`
	API      *syncAPIView         `json:"api,omitempty"`
	Import   syncImportView       `json:"import"`
	Export   exportView           `json:"export"`
	Relink   phoenix.RelinkReport `json:"relink"`
	Warnings []string             `json:"warnings"`
}

func newSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [export-dir]",
		Short: "Pull the MyMind library, mirror it to the vault, and re-link raw media",
		Long: "sync reads the library through the MyMind API when a key is configured\n" +
			"(CAIRN_MYMIND_KID and CAIRN_MYMIND_SECRET, or Keychain items mymind-api-kid\n" +
			"and mymind-api-secret); a full pull costs about one credit. Without a key, or\n" +
			"with --from-export, it reads the export folder's cards.csv instead. Either way\n" +
			"it then mirrors to the vault and re-links raw attachments. The export folder\n" +
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

			fromExport, _ := cmd.Flags().GetBool("from-export")
			dbPath := cairnDBPath()
			var (
				result importer.Result
				api    *syncAPIView
				source = exportDir
			)
			creds, credErr := mymindapi.LoadCredentials()
			switch {
			case fromExport || errors.Is(credErr, mymindapi.ErrNoCredentials):
				result, err = runImport(dbPath, exportDir, readSyncState(dbPath))
			case credErr != nil:
				return credErr
			default:
				source = mymindapi.DefaultBaseURL
				result, api, err = pullAndImport(cmd.Context(), mymindapi.NewClient(creds), dbPath, exportDir)
			}
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
				Source: source,
				API:    api,
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
	cmd.Flags().Bool("from-export", false, "Read the export folder's cards.csv even when an API key is configured")
	return cmd
}

// pullAndImport reads the library through the API, saving any missing
// attachment into exportDir so the importer and re-link treat it like an
// export file, then imports the snapshot.
func pullAndImport(ctx context.Context, client *mymindapi.Client, dbPath, exportDir string) (importer.Result, *syncAPIView, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return importer.Result{}, nil, err
	}
	pull, err := mymindapi.PullLibrary(ctx, client, exportDir)
	if err != nil {
		return importer.Result{}, nil, fmt.Errorf("pull from MyMind API: %w", err)
	}
	result, err := withDB(dbPath, readSyncState(dbPath), func(db *sql.DB) (importer.Result, error) {
		return importer.ImportCards(db, mymindapi.DefaultBaseURL, pull.Snapshot, exportDir, nil)
	})
	return result, &syncAPIView{
		Objects:            pull.Snapshot.RowsRead,
		Credits:            pull.Credits,
		SustainedRemaining: pull.Quota.SustainedRemaining,
		SustainedLimit:     pull.Quota.SustainedLimit,
		AttachmentsFetched: pull.BlobsFetched,
	}, err
}

func defaultRawExportDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "phoenix", "Clippings", "mymind")
}

func writeSyncPlain(out io.Writer, v syncView) error {
	var lines []string
	if v.API != nil {
		lines = append(lines, fmt.Sprintf("Pulled %d objects from the MyMind API: %s spent, %d of %d left this month; %d attachments fetched.",
			v.API.Objects, plural(v.API.Credits, "credit"), v.API.SustainedRemaining, v.API.SustainedLimit, v.API.AttachmentsFetched))
	}
	lines = append(lines,
		fmt.Sprintf("Imported %s: %d new, %d updated, %d unchanged, %d tombstoned.",
			v.Source, v.Import.Inserted, v.Import.Updated, v.Import.Unchanged, v.Import.Tombstoned),
		fmt.Sprintf("Mirrored to %s: %d written, %d unchanged, %d renamed; media %d written, %d already there.",
			v.Export.Path, v.Export.CardsWritten, v.Export.CardsUnchanged, v.Export.CardsRenamed, v.Export.MediaWritten, v.Export.MediaSkipped),
		fmt.Sprintf("Raw media: %d re-linked, %d already linked, %d without a vault copy.",
			v.Relink.Linked, v.Relink.AlreadyLinked, len(v.Relink.Missing)),
	)
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

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

package commands

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

type exportFile struct {
	Date string `json:"date"`
	File string `json:"file"`
}

type exportView struct {
	Cards     int          `json:"cards"`
	Files     []exportFile `json:"files"`
	Remaining int          `json:"remaining"`
	Media     struct {
		Count int    `json:"count"`
		Path  string `json:"path"`
	} `json:"media"`
	Path   string `json:"path"`
	DryRun bool   `json:"dry_run"`
}

type exportLine struct {
	Date       string `json:"date"`
	File       string `json:"file"`
	Remaining  int    `json:"remaining"`
	MediaCount int    `json:"media_count"`
	Path       string `json:"path"`
	MediaPath  string `json:"media_path"`
	DryRun     bool   `json:"dry_run"`
}

func newExportCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Mirror cards to Phoenix vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}
			limit, err := limitValue(cmd)
			if err != nil {
				return err
			}

			view := buildExportView(src, limit)
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(view))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL(exportLines(view)))
			default:
				err = writeExportPlain(out, view)
			}
			return err
		},
	}
	addListFlags(cmd)
	return cmd
}

func buildExportView(src source.Source, limit int) exportView {
	all := src.All()
	files := make([]exportFile, 0, len(all))
	for _, card := range all {
		files = append(files, exportFile{
			Date: card.CapturedAt.Format("2006-01-02"),
			File: slug(card.Title) + ".md",
		})
	}
	displayLimit := 3
	if limit > 0 {
		displayLimit = limit
	}
	limited := applyLimit(files, displayLimit)
	view := exportView{
		Cards:     len(all),
		Files:     limited,
		Remaining: len(files) - len(limited),
		Path:      "~/phoenix/Clippings/MyMind/",
		DryRun:    true,
	}
	view.Media.Count = countKind(all, cards.KindImage)
	view.Media.Path = "~/phoenix/Clippings/MyMind/_media/"
	return view
}

func writeExportPlain(out io.Writer, view exportView) error {
	if _, err := fmt.Fprintf(out, "Mirroring %d cards to %s\n\n", view.Cards, view.Path); err != nil {
		return err
	}
	for _, file := range view.Files {
		if _, err := fmt.Fprintf(out, "  %s  %s\n", file.Date, file.File); err != nil {
			return err
		}
	}
	if view.Remaining > 0 {
		if _, err := fmt.Fprintf(out, "  ... %d more files ...\n", view.Remaining); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "Media (%d files) to %s\n\n", view.Media.Count, view.Media.Path); err != nil {
		return err
	}
	_, err := fmt.Fprintln(out, "Phase 0: nothing written to disk. Real export runs in Phase 2.")
	return err
}

func exportLines(view exportView) []exportLine {
	lines := make([]exportLine, 0, len(view.Files))
	for _, file := range view.Files {
		lines = append(lines, exportLine{
			Date:       file.Date,
			File:       file.File,
			Remaining:  view.Remaining,
			MediaCount: view.Media.Count,
			Path:       view.Path,
			MediaPath:  view.Media.Path,
			DryRun:     view.DryRun,
		})
	}
	return lines
}

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	return strings.Trim(slugRe.ReplaceAllString(strings.ToLower(s), "-"), "-")
}

func countKind(all []cards.Card, k cards.Kind) int {
	n := 0
	for _, c := range all {
		if c.Kind == k {
			n++
		}
	}
	return n
}

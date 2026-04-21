package commands

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/fixtures"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Mirror cards to Phoenix vault",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			all := fixtures.All()
			fmt.Fprintf(out, "Mirroring %d cards to ~/phoenix/Clippings/MyMind/\n\n", len(all))
			for i, c := range all {
				if i < 3 {
					fmt.Fprintf(out, "  %s  %s.md\n", c.CapturedAt.Format("2006-01-02"), slug(c.Title))
				}
			}
			if len(all) > 3 {
				fmt.Fprintf(out, "  ... %d more files ...\n", len(all)-3)
			}
			fmt.Fprintln(out)
			mediaCount := countKind(all, cards.KindImage)
			fmt.Fprintf(out, "Media (%d files) to ~/phoenix/Clippings/MyMind/_media/\n", mediaCount)
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Phase 0: nothing written to disk. Real export runs in Phase 2.")
			return nil
		},
	}
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

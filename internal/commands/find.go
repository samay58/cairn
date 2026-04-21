package commands

import (
	"fmt"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/samay58/cairn/internal/render"
	"github.com/spf13/cobra"
)

func newFindCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "find",
		Short: "Full-screen fuzzy TUI (Phase 0: static mock)",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			all := fixtures.All()
			top := []render.Match{
				{Card: all[0], WhyShown: "recent"},
				{Card: all[1], WhyShown: "recent"},
				{Card: all[2], WhyShown: "recent"},
			}

			fmt.Fprintf(out, "cairn find · %d cards · type: all  source: all  since: all\n", len(all))
			fmt.Fprintln(out)
			fmt.Fprintln(out, "› _")
			fmt.Fprintln(out)
			fmt.Fprint(out, render.CardList(top))
			fmt.Fprintln(out)
			fmt.Fprintln(out, "enter open url · o open in mymind · c copy · y yank · tab cycle filters · esc quit")
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Phase 0: this is a static mock. Real TUI lands in Phase 2.")
			return nil
		},
	}
}

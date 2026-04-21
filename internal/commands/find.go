package commands

import (
	"fmt"

	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

func newFindCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "find",
		Short: "Full-screen fuzzy TUI (Phase 0: static mock)",
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}

			limit, err := limitValue(cmd)
			if err != nil {
				return err
			}

			top := applyLimit(findPreview(src), limit)
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.CardListJSON(top))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.CardListJSONL(top))
			default:
				_, err = fmt.Fprintf(out, "cairn find · %d cards · type: all  source: all  since: all\n\n", src.Count())
				if err != nil {
					return err
				}
				if _, err = fmt.Fprintln(out, "› _"); err != nil {
					return err
				}
				if _, err = fmt.Fprintln(out); err != nil {
					return err
				}
				if _, err = fmt.Fprint(out, render.CardList(top)); err != nil {
					return err
				}
				if _, err = fmt.Fprintln(out); err != nil {
					return err
				}
				if _, err = fmt.Fprintln(out, "enter open url · o open in mymind · c copy · y yank"); err != nil {
					return err
				}
				if _, err = fmt.Fprintln(out, "tab cycle filters · esc quit"); err != nil {
					return err
				}
				if _, err = fmt.Fprintln(out); err != nil {
					return err
				}
				_, err = fmt.Fprintln(out, "Phase 0: this is a static mock. Real TUI lands in Phase 2.")
			}
			return err
		},
	}
	addListFlags(cmd)
	return cmd
}

func findPreview(src source.Source) []render.Match {
	all := src.All()
	return []render.Match{
		{Card: all[0], WhyShown: "Recent."},
		{Card: all[1], WhyShown: "Recent."},
		{Card: all[2], WhyShown: "Recent."},
	}
}

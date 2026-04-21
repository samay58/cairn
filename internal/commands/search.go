package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

func newSearchCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Hybrid retrieval, card-list output",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}

			limit, err := limitValue(cmd)
			if err != nil {
				return err
			}

			query := strings.Join(args, " ")
			matches := src.Search(query, source.Filters{}, limit)
			if err := src.LastListSave(matches); err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: failed to persist handles: %v\n", err)
			}

			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.CardListJSON(matches))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.CardListJSONL(matches))
			default:
				if len(matches) == 0 {
					return writeNoResults(out, query)
				}
				_, err = fmt.Fprint(out, render.CardList(matches))
			}
			return err
		},
	}
	addListFlags(cmd)
	return cmd
}

func writeNoResults(out io.Writer, query string) error {
	if _, err := fmt.Fprintf(out, "No cards matched %q.\n\n", query); err != nil {
		return err
	}
	for _, line := range render.WrapLines("", "Try a broader query, drop filters, or check `cairn status` to confirm the import is fresh.", render.DefaultWidth) {
		if _, err := fmt.Fprintf(out, "%s\n", line); err != nil {
			return err
		}
	}
	return nil
}

package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/samay58/cairn/internal/render"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Hybrid retrieval, card-list output",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			matches := fakeSearch(query)

			limit, _ := cmd.Flags().GetInt("limit")
			if limit > 0 && limit < len(matches) {
				matches = matches[:limit]
			}

			asJSON, _ := cmd.Flags().GetBool("json")
			asJSONL, _ := cmd.Flags().GetBool("jsonl")

			out := cmd.OutOrStdout()
			switch {
			case asJSON:
				fmt.Fprint(out, render.CardListJSON(matches))
			case asJSONL:
				fmt.Fprint(out, render.CardListJSONL(matches))
			default:
				if len(matches) == 0 {
					return writeNoResults(out, query)
				}
				fmt.Fprint(out, render.CardList(matches))
			}
			return nil
		},
	}
}

func fakeSearch(query string) []render.Match {
	q := strings.ToLower(strings.TrimSpace(query))
	all := fixtures.All()
	switch q {
	case "oauth":
		return []render.Match{
			{Card: all[0], WhyShown: "matched on title and tag oauth"},
			{Card: all[10], WhyShown: "matched on tag auth"},
			{Card: all[17], WhyShown: "matched on body"},
		}
	case "zzz-empty":
		return nil
	}
	return []render.Match{
		{Card: all[0], WhyShown: "demo result 1"},
		{Card: all[1], WhyShown: "demo result 2"},
		{Card: all[2], WhyShown: "demo result 3"},
	}
}

func writeNoResults(out io.Writer, query string) error {
	_, err := fmt.Fprintf(out,
		"No cards matched %q.\n\nTry a broader query, drop filters, or check `cairn status` to confirm the import is fresh.\n",
		query,
	)
	return err
}

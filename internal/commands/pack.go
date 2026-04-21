package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/fixtures"
	"github.com/spf13/cobra"
)

func newPackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack <query>",
		Short: "Cited context pack for an external AI",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			profile, _ := cmd.Flags().GetString("for")
			picks := packPicks(query)
			out := cmd.OutOrStdout()
			switch profile {
			case "claude", "":
				writePackClaude(out, query, picks)
			case "json":
				writePackJSON(out, query, picks)
			default:
				fmt.Fprintf(out, "Unknown profile %q. Supported in Phase 0: claude, json.\n", profile)
			}
			return nil
		},
	}
	cmd.Flags().String("for", "claude", "target profile: claude, chatgpt, cursor, markdown, json")
	return cmd
}

func packPicks(query string) []cards.Card {
	all := fixtures.All()
	// Phase 0 deterministic subset for the oauth device flow query.
	return []cards.Card{all[0], all[10], all[17]}
}

func writePackClaude(out io.Writer, query string, picks []cards.Card) {
	fmt.Fprintf(out, "<context source=\"cairn\" query=%q cards=\"%d\">\n", query, len(picks))
	for _, c := range picks {
		fmt.Fprintf(out, "  <card id=%q kind=%q captured=%q source=%q>\n",
			c.ID, string(c.Kind), c.CapturedAt.Format("2006-01-02"), c.Source)
		fmt.Fprintf(out, "    <title>%s</title>\n", c.Title)
		if c.URL != "" {
			fmt.Fprintf(out, "    <url>%s</url>\n", c.URL)
		}
		excerpt := c.Excerpt
		if excerpt == "" {
			excerpt = c.Body
		}
		fmt.Fprintf(out, "    <excerpt>%s</excerpt>\n", excerpt)
		fmt.Fprintln(out, "  </card>")
	}
	fmt.Fprintln(out, "</context>")
	fmt.Fprintln(out)
	keys := make([]string, 0, len(picks))
	for _, c := range picks {
		keys = append(keys, c.ID)
	}
	fmt.Fprintf(out, "Citation key: %s.\n", strings.Join(keys, ", "))
}

func writePackJSON(out io.Writer, query string, picks []cards.Card) {
	payload := map[string]any{
		"query":         query,
		"cards":         picks,
		"citation_keys": citationKeys(picks),
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	fmt.Fprintln(out, string(b))
}

func citationKeys(picks []cards.Card) []string {
	out := make([]string, 0, len(picks))
	for _, c := range picks {
		out = append(out, c.ID)
	}
	return out
}

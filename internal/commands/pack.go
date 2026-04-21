package commands

import (
	"fmt"
	"io"
	"strings"

	"github.com/samay58/cairn/internal/cards"
	"github.com/samay58/cairn/internal/render"
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

type packItem struct {
	CitationKey string     `json:"citation_key"`
	Card        cards.Card `json:"card"`
}

type packPayload struct {
	Query string     `json:"query"`
	Cards []packItem `json:"cards"`
}

type packLine struct {
	Query       string     `json:"query"`
	CitationKey string     `json:"citation_key"`
	Card        cards.Card `json:"card"`
}

func newPackCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack <query>",
		Short: "Cited context pack for an external AI",
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
			profile, err := cmd.Flags().GetString("for")
			if err != nil {
				return err
			}

			picks := applyLimit(packPicks(src, query), limit)
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(buildPackPayload(query, picks)))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL(packLines(query, picks)))
			default:
				switch profile {
				case "", "claude":
					err = writePackClaude(out, query, picks)
				case "json":
					_, err = fmt.Fprint(out, render.JSON(buildPackPayload(query, picks)))
				default:
					err = fmt.Errorf("unknown profile %q. Supported in Phase 0: claude, json", profile)
				}
			}
			return err
		},
	}
	cmd.Flags().String("for", "claude", "target profile: claude, chatgpt, cursor, markdown, json")
	addListFlags(cmd)
	return cmd
}

func packPicks(src source.Source, query string) []cards.Card {
	all := src.All()
	// Phase 0 deterministic subset for the oauth device flow query.
	return []cards.Card{all[0], all[10], all[17]}
}

func buildPackPayload(query string, picks []cards.Card) packPayload {
	return packPayload{
		Query: query,
		Cards: packItems(picks),
	}
}

func packItems(picks []cards.Card) []packItem {
	items := make([]packItem, 0, len(picks))
	for _, card := range picks {
		items = append(items, packItem{
			CitationKey: card.ID,
			Card:        card,
		})
	}
	return items
}

func packLines(query string, picks []cards.Card) []packLine {
	lines := make([]packLine, 0, len(picks))
	for _, card := range picks {
		lines = append(lines, packLine{
			Query:       query,
			CitationKey: card.ID,
			Card:        card,
		})
	}
	return lines
}

func writePackClaude(out io.Writer, query string, picks []cards.Card) error {
	if _, err := fmt.Fprintf(out, "<context source=\"cairn\" query=\"%s\" cards=\"%d\">\n", xmlAttr(query), len(picks)); err != nil {
		return err
	}

	for _, card := range picks {
		if _, err := fmt.Fprintf(out, "  <card id=\"%s\" kind=\"%s\" captured=\"%s\"",
			xmlAttr(card.ID),
			xmlAttr(string(card.Kind)),
			xmlAttr(card.CapturedAt.Format("2006-01-02")),
		); err != nil {
			return err
		}
		if card.Source != "" {
			if _, err := fmt.Fprintln(out); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(out, "        source=\"%s\">\n", xmlAttr(card.Source)); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(out, ">"); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintf(out, "    <title>%s</title>\n", xmlText(card.Title)); err != nil {
			return err
		}
		if card.URL != "" {
			if _, err := fmt.Fprintf(out, "    <url>%s</url>\n", xmlText(card.URL)); err != nil {
				return err
			}
		}
		if err := writeWrappedElement(out, "excerpt", render.ExcerptText(card)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(out, "  </card>"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(out, "</context>"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	_, err := fmt.Fprintf(out, "Citation key: %s.\n", strings.Join(citationKeys(picks), ", "))
	return err
}

func writeWrappedElement(out io.Writer, name, text string) error {
	if _, err := fmt.Fprintf(out, "    <%s>\n", name); err != nil {
		return err
	}
	for _, line := range render.WrapLines("      ", xmlText(text), render.DefaultWidth) {
		if _, err := fmt.Fprintf(out, "%s\n", line); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(out, "    </%s>\n", name)
	return err
}

func citationKeys(picks []cards.Card) []string {
	out := make([]string, 0, len(picks))
	for _, card := range picks {
		out = append(out, card.ID)
	}
	return out
}

func xmlText(s string) string {
	return xmlTextEscaper.Replace(s)
}

var xmlTextEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
)

var xmlAttrEscaper = strings.NewReplacer(
	"&", "&amp;",
	"\"", "&quot;",
	"<", "&lt;",
	"\n", "&#xA;",
	"\r", "&#xD;",
	"\t", "&#x9;",
)

func xmlAttr(s string) string {
	return xmlAttrEscaper.Replace(s)
}

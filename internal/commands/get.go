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

type getView struct {
	Handle int        `json:"handle"`
	Card   cards.Card `json:"card"`
}

func newGetCmd(src source.Source) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <card>",
		Short: "Render full card in terminal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := selectedOutputMode(cmd)
			if err != nil {
				return err
			}

			n, err := parseHandle(args[0])
			if err != nil {
				return err
			}

			card, err := src.ByHandle(n)
			if err != nil {
				return fmt.Errorf("%w\nRun a list command (search, find) to refresh handles.", err)
			}

			view := getView{Handle: n, Card: card}
			out := cmd.OutOrStdout()
			switch mode {
			case outputJSON:
				_, err = fmt.Fprint(out, render.JSON(view))
			case outputJSONL:
				_, err = fmt.Fprint(out, render.JSONL([]getView{view}))
			default:
				err = writeGetPlain(out, view)
			}
			return err
		},
	}
	addOutputFlags(cmd)
	return cmd
}

func writeGetPlain(out io.Writer, view getView) error {
	if _, err := fmt.Fprintf(out, "@%d  %s\n", view.Handle, view.Card.Title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(out, "%s\n", render.MetaLine(view.Card)); err != nil {
		return err
	}
	if len(view.Card.Tags) > 0 {
		if _, err := fmt.Fprintf(out, "tags: %s\n", strings.Join(view.Card.Tags, ", ")); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(out); err != nil {
		return err
	}
	for _, line := range render.WrapLines("", render.ExcerptText(view.Card), render.DefaultWidth) {
		if _, err := fmt.Fprintf(out, "%s\n", line); err != nil {
			return err
		}
	}
	return nil
}

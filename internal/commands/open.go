package commands

import (
	"fmt"

	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

func newOpenCmd(src source.Source) *cobra.Command {
	return &cobra.Command{
		Use:   "open <card>",
		Short: "Open a card in the MyMind browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := parseHandle(args[0])
			if err != nil {
				return err
			}

			card, err := src.ByHandle(n)
			if err != nil {
				return fmt.Errorf("%w\nRun a list command (search, find) to refresh handles.", err)
			}

			url := card.URL
			if url == "" {
				url = fmt.Sprintf("https://access.mymind.com/cards/%s", card.MyMindID)
			}

			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Would open: %s\n", url); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "Phase 0 fake. Real `cairn open` invokes the OS default browser.")
			return err
		},
	}
}

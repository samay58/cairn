package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open <card>",
		Short: "Open a card in the MyMind browser",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			n, err := strconv.Atoi(strings.TrimPrefix(args[0], "@"))
			if err != nil {
				fmt.Fprintf(out, "Invalid handle: %q.\n", args[0])
				return nil
			}
			c, err := fixtures.ByHandle(n)
			if err != nil {
				fmt.Fprintln(out, err.Error())
				return nil
			}
			url := c.URL
			if url == "" {
				url = fmt.Sprintf("https://access.mymind.com/cards/%s", c.MyMindID)
			}
			fmt.Fprintf(out, "Would open: %s\n", url)
			fmt.Fprintln(out, "(Phase 0 fake: real `cairn open` invokes the OS default browser.)")
			return nil
		},
	}
}

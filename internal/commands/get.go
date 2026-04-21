package commands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/samay58/cairn/internal/fixtures"
	"github.com/samay58/cairn/internal/render"
	"github.com/spf13/cobra"
)

func newGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <card>",
		Short: "Render full card in terminal",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			ref := args[0]
			if !strings.HasPrefix(ref, "@") {
				fmt.Fprintf(out, "Phase 0 only supports @handle references. Got %q.\n", ref)
				return nil
			}
			n, err := strconv.Atoi(strings.TrimPrefix(ref, "@"))
			if err != nil {
				fmt.Fprintf(out, "Invalid handle: %q.\n", ref)
				return nil
			}
			c, err := fixtures.ByHandle(n)
			if err != nil {
				fmt.Fprintln(out, err.Error())
				fmt.Fprintln(out, "Run a list command (search, find) to refresh handles.")
				return nil
			}
			meta := render.MetaLine(c)
			fmt.Fprintf(out, "@%d  %s\n", n, c.Title)
			fmt.Fprintln(out, meta)
			if len(c.Tags) > 0 {
				fmt.Fprintf(out, "tags: %s\n", strings.Join(c.Tags, ", "))
			}
			fmt.Fprintln(out)
			body := c.Body
			if body == "" {
				body = c.Excerpt
			}
			fmt.Fprintln(out, body)
			return nil
		},
	}
}

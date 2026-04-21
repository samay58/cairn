package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

func newOpenCmd(src source.Source) *cobra.Command {
	return &cobra.Command{
		Use:   "open <card>",
		Short: "Open a card in the default browser",
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

			out := cmd.OutOrStdout()
			if os.Getenv("CAIRN_DRY_OPEN") == "1" {
				_, err := fmt.Fprintf(out, "Would open: %s\n", url)
				return err
			}
			return launchBrowser(out, url)
		},
	}
}

func launchBrowser(out io.Writer, url string) error {
	var c *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		c = exec.Command("open", url)
	case "linux":
		c = exec.Command("xdg-open", url)
	case "windows":
		c = exec.Command("cmd", "/c", "start", url)
	default:
		_, err := fmt.Fprintf(out, "Open this URL manually: %s\n", url)
		return err
	}
	if err := c.Start(); err != nil {
		return fmt.Errorf("open card: launch browser: %w\nurl: %s", err, url)
	}
	_, err := fmt.Fprintf(out, "Opened: %s\n", url)
	return err
}

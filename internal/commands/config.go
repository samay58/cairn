package commands

import (
	_ "embed"

	"github.com/spf13/cobra"
)

//go:embed config_stub.txt
var configStub string

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Edit config (Phase 0: shows defaults only)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := cmd.OutOrStdout().Write([]byte(configStub))
			return err
		},
	}
}

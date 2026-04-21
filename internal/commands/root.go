package commands

import (
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	return NewRootWithSource(source.NewFixtureSource())
}

func NewRootWithSource(src source.Source) *cobra.Command {
	root := &cobra.Command{
		Use:           "cairn",
		Short:         "Terminal-native bridge between MyMind and the tools you already use",
		Long:          "Cairn makes your MyMind library queryable from the terminal and a first-class context source for AI tools.\n\nPhase 0 is a design prototype with hand-authored output. Real storage and search arrive in Phase 1.",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	root.AddCommand(
		newImportCmd(),
		newStatusCmd(src),
		newSearchCmd(src),
		newFindCmd(src),
		newGetCmd(src),
		newOpenCmd(src),
		newPackCmd(src),
		newAskCmd(),
		newExportCmd(src),
		newConfigCmd(),
		newMCPCmd(),
	)
	return root
}

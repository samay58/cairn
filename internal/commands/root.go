package commands

import (
	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

// Execute runs the cairn CLI with its default Source (SQLite if the database
// exists at the configured path, else fixtures).
func Execute() error {
	src, err := openSource(cairnDBPath())
	if err != nil {
		return err
	}
	return NewRootWithSource(src).Execute()
}

// NewRoot keeps the Phase 0 constructor intact for tests that build the tree
// explicitly with a FixtureSource.
func NewRoot() *cobra.Command {
	return NewRootWithSource(source.NewFixtureSource())
}

func NewRootWithSource(src source.Source) *cobra.Command {
	fixture := source.NewFixtureSource()
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
		newFindCmd(fixture),
		newGetCmd(src),
		newOpenCmd(src),
		newPackCmd(fixture),
		newAskCmd(),
		newExportCmd(fixture),
		newConfigCmd(),
		newMCPCmd(),
	)
	return root
}

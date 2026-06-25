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
	root := &cobra.Command{
		Use:           "cairn",
		Short:         "Terminal-native bridge between MyMind and the tools you already use",
		Long:          "Cairn makes your MyMind library queryable from the terminal and mirrors it into Phoenix markdown.\n\nReal today: import, status, search, get, open, export. Designed but not implemented yet: find, pack, ask, mcp.",
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

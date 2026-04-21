package commands

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/samay58/cairn/internal/source"
	"github.com/spf13/cobra"
)

// importSampleHelper runs the real import command against the sample export
// dir, writing the resulting DB under CAIRN_HOME.
func importSampleHelper(t *testing.T) {
	t.Helper()
	root := NewRootWithSource(source.NewFixtureSource())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"import", filepath.Join("..", "..", "testdata", "mymind_sample_export")})
	if err := root.Execute(); err != nil {
		t.Fatalf("import helper failed: %v\n%s", err, buf.String())
	}
}

// buildRootForCurrentDB constructs a root command backed by the SQLite source
// at cairnDBPath(). Falls back to a FixtureSource if the DB is missing.
func buildRootForCurrentDB() (*cobra.Command, error) {
	src, err := openSource(cairnDBPath())
	if err != nil {
		return nil, err
	}
	return NewRootWithSource(src), nil
}

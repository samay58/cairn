package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
	"github.com/samay58/cairn/internal/source"
)

func TestOpenByHandle(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	t.Setenv("CAIRN_DRY_OPEN", "1")

	root := NewRootWithSource(source.NewFixtureSource())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"open", "@1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "open_at1.txt", out.String())
}

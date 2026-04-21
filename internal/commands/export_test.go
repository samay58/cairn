package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestExport(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"export"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "export.txt", out.String())
}

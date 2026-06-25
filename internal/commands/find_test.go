package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestFindFrame(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"find"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "find_frame.txt", out.String())
}

func TestFindImportedLibraryIsNotImplemented(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	importSampleHelper(t)

	root, err := buildRootForCurrentDB()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"find"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "`cairn find` is not implemented for imported libraries yet.") {
		t.Fatalf("expected not implemented message, got:\n%s", out.String())
	}
}

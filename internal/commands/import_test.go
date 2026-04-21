package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestImportOK(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"import", "/tmp/mymind-export-2026-04-19/"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "import_ok.txt", out.String())
}

func TestImportNotFound(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SilenceErrors = true
	root.SilenceUsage = true
	root.SetArgs([]string{"import", "/tmp/does-not-exist"})
	_ = root.Execute()
	golden.Assert(t, "import_err_notfound.txt", out.String())
}

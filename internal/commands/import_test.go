package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/golden"
	"github.com/samay58/cairn/internal/source"
)

func TestImportSampleExport(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CAIRN_HOME", tmp)

	exportDir := filepath.Join("..", "..", "testdata", "mymind_sample_export")

	src, err := openSource(filepath.Join(tmp, "cairn.db"))
	if err != nil {
		t.Fatal(err)
	}
	// Use the fallback source for help wiring; the import command opens its own DB.
	_ = src

	root := NewRootWithSource(source.NewFixtureSource())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"import", exportDir})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	outStr := strings.ReplaceAll(out.String(), tmp, "<TMP>")
	golden.Assert(t, "import_ok_phase1.txt", outStr)

	// Confirm the DB was actually created.
	if _, err := os.Stat(filepath.Join(tmp, "cairn.db")); err != nil {
		t.Fatalf("DB not written: %v", err)
	}
}

func TestImportNotFound(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())

	root := NewRootWithSource(source.NewFixtureSource())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&out)
	root.SetArgs([]string{"import", "/tmp/does-not-exist-x9z9z"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "import_err_notfound_phase1.txt", out.String())
}

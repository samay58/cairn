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

	dbPath := filepath.Join(tmp, "cairn.db")
	outStr := strings.ReplaceAll(out.String(), dbPath, "<TMP>/cairn.db")
	outStr = strings.ReplaceAll(outStr, strings.Join(wrapFilesystemPath("  ", dbPath, 80), "\n"), "  <TMP>/cairn.db")
	golden.Assert(t, "import_ok_phase1.txt", outStr)

	// Confirm the DB was actually created.
	if _, err := os.Stat(filepath.Join(tmp, "cairn.db")); err != nil {
		t.Fatalf("DB not written: %v", err)
	}
}

func TestImportNotFound(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())

	root := NewRootWithSource(source.NewFixtureSource())
	root.SetArgs([]string{"import", "/tmp/does-not-exist-x9z9z"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected import error")
	}
	golden.Assert(t, "import_err_notfound_phase1.txt", err.Error()+"\n")
}

func TestImportSampleExportLongHomeFitsWidth(t *testing.T) {
	base := t.TempDir()
	longHome := filepath.Join(base,
		"phase-one-review",
		"long-output-contract-check",
		"nested-home-for-cairn",
		"library-state")
	t.Setenv("CAIRN_HOME", longHome)

	root := NewRootWithSource(source.NewFixtureSource())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"import", filepath.Join("..", "..", "testdata", "mymind_sample_export")})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	assertFitsDefaultWidth(t, out.String())
}

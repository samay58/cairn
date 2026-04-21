package commands

import (
	"bytes"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/golden"
	"github.com/samay58/cairn/internal/source"
)

var timestampRE = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)
var sizeRE = regexp.MustCompile(`\d+(\.\d+)? (B|KB|MB)`)

func TestStatusNoDatabase(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())

	root := NewRootWithSource(source.NewFixtureSource())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "status_nodatabase.txt", out.String())
}

func TestStatusImported(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CAIRN_HOME", tmp)

	// Import the sample to populate the DB.
	importSampleHelper(t)

	root, err := buildRootForCurrentDB()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetErr(&bytes.Buffer{})
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tmp, "cairn.db")
	got := strings.ReplaceAll(out.String(), dbPath, "<TMP>/cairn.db")
	got = strings.ReplaceAll(got, strings.Join(wrapFilesystemPath("          ", dbPath, 80), "\n"), "          <TMP>/cairn.db")
	got = timestampRE.ReplaceAllString(got, "<TS>")
	got = sizeRE.ReplaceAllString(got, "<SIZE>")
	golden.Assert(t, "status_imported.txt", got)
}

func TestStatusFailedImport(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("CAIRN_HOME", tmp)

	root := NewRootWithSource(source.NewFixtureSource())
	var importOut bytes.Buffer
	root.SetOut(&importOut)
	root.SetErr(&importOut)
	root.SetArgs([]string{"import", t.TempDir()})
	if err := root.Execute(); err == nil {
		t.Fatal("expected failed import")
	}

	root, err := buildRootForCurrentDB()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(tmp, "cairn.db")
	got := strings.ReplaceAll(out.String(), dbPath, "<TMP>/cairn.db")
	got = strings.ReplaceAll(got, strings.Join(wrapFilesystemPath("          ", dbPath, 80), "\n"), "          <TMP>/cairn.db")
	got = timestampRE.ReplaceAllString(got, "<TS>")
	got = sizeRE.ReplaceAllString(got, "<SIZE>")
	golden.Assert(t, "status_failed_import.txt", got)
}

func TestStatusImportedLongHomeFitsWidth(t *testing.T) {
	base := t.TempDir()
	longHome := filepath.Join(base,
		"phase-one-review",
		"long-output-contract-check",
		"nested-home-for-cairn",
		"library-state")
	t.Setenv("CAIRN_HOME", longHome)

	importSampleHelper(t)

	root, err := buildRootForCurrentDB()
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	assertFitsDefaultWidth(t, out.String())
}

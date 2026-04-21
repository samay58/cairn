package commands

import (
	"bytes"
	"regexp"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/golden"
	"github.com/samay58/cairn/internal/source"
)

var timestampRE = regexp.MustCompile(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z`)
var sizeRE = regexp.MustCompile(`\(\d+(\.\d+)? (B|KB|MB)\)`)

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
	root.SetArgs([]string{"status"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	got := strings.ReplaceAll(out.String(), tmp, "<TMP>")
	got = timestampRE.ReplaceAllString(got, "<TS>")
	got = sizeRE.ReplaceAllString(got, "(<SIZE>)")
	golden.Assert(t, "status_imported.txt", got)
}

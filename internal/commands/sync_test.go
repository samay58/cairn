package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/samay58/cairn/internal/source"
)

func runSync(t *testing.T, raw, vault string) syncView {
	t.Helper()
	root := NewRootWithSource(source.NewFixtureSource())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"sync", raw, "--to", vault, "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("sync failed: %v\n%s", err, buf.String())
	}
	var v syncView
	if err := json.Unmarshal(buf.Bytes(), &v); err != nil {
		t.Fatalf("decode sync json: %v\n%s", err, buf.String())
	}
	return v
}

func TestSyncImportsMirrorsAndRelinks(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	raw := filepath.Join(t.TempDir(), "mymind")
	copySampleExport(t, raw)
	vault := filepath.Join(t.TempDir(), "mymind-cards")

	first := runSync(t, raw, vault)
	if first.Import.Inserted == 0 || first.Export.CardsWritten != first.Import.Inserted {
		t.Fatalf("first sync = %+v; want every inserted card written", first)
	}
	if first.Relink.Linked != 1 || len(first.Relink.Missing) != 0 {
		t.Fatalf("relink = %+v; want the one sample image linked", first.Relink)
	}
	if fi, err := os.Lstat(filepath.Join(raw, "media", "mm_1.png")); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("raw image should now be a symlink into the vault")
	}

	// A second run over the linked folder changes nothing.
	second := runSync(t, raw, vault)
	if second.Import.Inserted != 0 || second.Import.Updated != 0 || second.Export.CardsWritten != 0 {
		t.Errorf("second sync = %+v; want no changes", second)
	}
	if second.Relink.Linked != 0 || second.Relink.AlreadyLinked != 1 {
		t.Errorf("second relink = %+v; want 1 already linked", second.Relink)
	}
}

func TestSyncRefusesVaultInsideExportPath(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	raw := filepath.Join(t.TempDir(), "mymind")
	copySampleExport(t, raw)
	root := NewRootWithSource(source.NewFixtureSource())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"sync", raw, "--to", raw})
	if err := root.Execute(); err == nil {
		t.Fatal("sync into the export folder should fail")
	}
}

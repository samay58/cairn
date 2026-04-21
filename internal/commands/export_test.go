package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupImportedRoot imports testdata/mymind_sample_export into a temp
// CAIRN_HOME and returns a command root bound to that SQLite source.
func setupImportedRoot(t *testing.T) *bytes.Buffer {
	t.Helper()
	home := t.TempDir()
	t.Setenv("CAIRN_HOME", home)
	importSampleHelper(t)
	buf := &bytes.Buffer{}
	return buf
}

func runExport(t *testing.T, vault string, extra ...string) string {
	t.Helper()
	root, err := buildRootForCurrentDB()
	if err != nil {
		t.Fatalf("build root: %v", err)
	}
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	args := append([]string{"export", "--to", vault}, extra...)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		t.Fatalf("export failed: %v\n%s", err, buf.String())
	}
	return buf.String()
}

func normalizeVault(out, vault string) string {
	return strings.ReplaceAll(out, vault, "<VAULT>")
}

func TestExportDryRunWritesNothing(t *testing.T) {
	setupImportedRoot(t)
	vault := t.TempDir()
	got := runExport(t, vault, "--dry-run")
	got = normalizeVault(got, vault)
	const want = "Would write 4 cards to <VAULT>\n  media: 1 written, 0 skipped\nRemove --dry-run to write.\n"
	if got != want {
		t.Fatalf("dry-run output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	entries, _ := os.ReadDir(vault)
	if len(entries) != 0 {
		t.Fatalf("dry-run wrote %d entries, want 0", len(entries))
	}
}

func TestExportRealWritesMarkdown(t *testing.T) {
	setupImportedRoot(t)
	vault := t.TempDir()
	got := runExport(t, vault)
	got = normalizeVault(got, vault)
	if !strings.HasPrefix(got, "Wrote 4 cards to <VAULT>") {
		t.Fatalf("unexpected output:\n%s", got)
	}
	matches, _ := filepath.Glob(filepath.Join(vault, "*.md"))
	if len(matches) != 4 {
		t.Fatalf("expected 4 markdown files in vault, got %d", len(matches))
	}
	// Media got linked in via import; verify at least one _media file exists.
	mediaGlobs, _ := filepath.Glob(filepath.Join(vault, "_media", "*", "*", "*"))
	if len(mediaGlobs) < 1 {
		t.Fatalf("expected at least one media file under _media/, got %d", len(mediaGlobs))
	}
}

func TestExportSecondRunIsUnchanged(t *testing.T) {
	setupImportedRoot(t)
	vault := t.TempDir()
	_ = runExport(t, vault)
	got := runExport(t, vault)
	got = normalizeVault(got, vault)
	if !strings.Contains(got, "4 cards unchanged") {
		t.Fatalf("expected 'cards unchanged' on re-run, got:\n%s", got)
	}
}

func TestExportFreshInstallRefusesWithoutImport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CAIRN_HOME", home)
	root, err := buildRootForCurrentDB()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	vault := t.TempDir()
	root.SetArgs([]string{"export", "--to", vault})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "No import recorded yet") {
		t.Fatalf("expected refusal message, got:\n%s", buf.String())
	}
	entries, _ := os.ReadDir(vault)
	if len(entries) != 0 {
		t.Fatalf("fresh install wrote %d entries, want 0", len(entries))
	}
}

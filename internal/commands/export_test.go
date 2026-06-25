package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/source"
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

func TestDefaultExportRootUsesKnowledgeBaseMirror(t *testing.T) {
	root := defaultExportRoot()
	wantSuffix := filepath.Join("phoenix", "04-knowledge-base", "research-archive", "mymind-cards")
	if !strings.HasSuffix(root, wantSuffix) {
		t.Fatalf("default export root = %q, want suffix %q", root, wantSuffix)
	}
}

func TestExportRefusesCaseInsensitiveImportPathCollision(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CAIRN_HOME", home)

	parent := filepath.Join(t.TempDir(), "Clippings")
	importDir := filepath.Join(parent, "mymind")
	if err := os.MkdirAll(importDir, 0o755); err != nil {
		t.Fatal(err)
	}
	copySampleExport(t, importDir)

	root := NewRootWithSource(source.NewFixtureSource())
	root.SetArgs([]string{"import", importDir})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}

	root, err := buildRootForCurrentDB()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"export", "--to", filepath.Join(parent, "MyMind")})
	err = root.Execute()
	if err == nil {
		t.Fatalf("expected collision error, got output:\n%s", buf.String())
	}
	if !strings.Contains(err.Error(), "export target matches the last import path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func copySampleExport(t *testing.T, dest string) {
	t.Helper()
	src := filepath.Join("..", "..", "testdata", "mymind_sample_export")
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dest, entry.Name()), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

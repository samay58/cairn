package phoenix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelinkRawReplacesCopiesWithLinks(t *testing.T) {
	raw := filepath.Join(t.TempDir(), "mymind")
	root := t.TempDir()
	if err := os.MkdirAll(raw, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(p string, b []byte) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	pdf := filepath.Join(raw, "MID1.pdf")
	write(pdf, []byte("%PDF-1.7 attachment"))
	canonical := filepath.Join(root, MediaRelPath(contentSHA(t, pdf), "pdf"))
	write(canonical, []byte("%PDF-1.7 attachment"))
	orphan := filepath.Join(raw, "MID2.jpeg")
	write(orphan, []byte("no canonical copy"))
	write(filepath.Join(raw, "cards.csv"), []byte("id,title\n"))

	rep, err := RelinkRaw(raw, root)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Linked != 1 || rep.AlreadyLinked != 0 || len(rep.Missing) != 1 || rep.Missing[0] != orphan {
		t.Fatalf("report = %+v, want 1 linked and %s missing", rep, orphan)
	}
	target, err := os.Readlink(pdf)
	if err != nil {
		t.Fatalf("%s is not a symlink: %v", pdf, err)
	}
	if filepath.IsAbs(target) {
		t.Errorf("link target %q is absolute; want relative so the vault can move", target)
	}
	if got := contentSHA(t, pdf); got != contentSHA(t, canonical) {
		t.Errorf("link resolves to sha %s, want the canonical copy", got)
	}
	if fi, err := os.Lstat(orphan); err != nil || !fi.Mode().IsRegular() {
		t.Errorf("orphan should stay a regular file")
	}

	again, err := RelinkRaw(raw, root)
	if err != nil {
		t.Fatal(err)
	}
	if again.Linked != 0 || again.AlreadyLinked != 1 {
		t.Errorf("second run = %+v, want 0 linked and 1 already linked", again)
	}
}

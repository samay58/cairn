package golden

import (
	"os"
	"path/filepath"
	"testing"
)

// Assert compares got to the contents of testdata/golden/<name>.
// If UPDATE_GOLDEN=1, writes got to disk and passes.
func Assert(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %q not found: %v (run with UPDATE_GOLDEN=1 to create)", path, err)
	}
	if string(want) != got {
		t.Errorf("golden %q mismatch\n--- want ---\n%s\n--- got ---\n%s", path, string(want), got)
	}
}

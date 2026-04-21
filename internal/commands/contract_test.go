package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTextGoldensRespectOutputContract(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("testdata", "golden", "*.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		trimmed := strings.TrimSpace(string(data))
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			continue
		}
		lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
		for i, line := range lines {
			if utf8.RuneCountInString(line) > 80 {
				t.Fatalf("%s:%d exceeds 80 columns", path, i+1)
			}
			if strings.Contains(line, "—") {
				t.Fatalf("%s:%d contains em dash", path, i+1)
			}
		}
	}
}

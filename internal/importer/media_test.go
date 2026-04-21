package importer

import (
	"path/filepath"
	"testing"
)

func TestScanMediaHashesAndDetectsMime(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "mymind_sample_export", "media")
	items, err := ScanMedia(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d media items, want 1", len(items))
	}
	if items[0].Mime != "image/png" {
		t.Errorf("mime = %q, want image/png", items[0].Mime)
	}
	if len(items[0].SHA256) != 64 {
		t.Errorf("sha256 = %q (len %d), want 64 hex chars", items[0].SHA256, len(items[0].SHA256))
	}
}

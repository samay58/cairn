package cards

import "testing"

func TestMediaZeroValue(t *testing.T) {
	var m Media
	if m.Kind != "" || m.Path != "" || m.SHA256 != "" || m.Mime != "" {
		t.Fatalf("expected zero Media, got %+v", m)
	}
}

func TestMediaFieldsExported(t *testing.T) {
	m := Media{Kind: "document", Path: "x.pdf", SHA256: "abc", Mime: "application/pdf"}
	if m.Kind != "document" || m.Mime != "application/pdf" {
		t.Fatalf("fields did not round-trip")
	}
}

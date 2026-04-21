package phoenix

import (
	"os"
	"path/filepath"
	"strings"
)

func (w *Writer) Write(bundles []CardBundle) (WriteReport, error) {
	var r WriteReport
	if !w.DryRun {
		if err := os.MkdirAll(w.Root, 0o755); err != nil {
			return r, err
		}
	}
	writtenThisBatch := make(map[string]bool)
	for _, b := range bundles {
		name := w.resolveFilename(b.Card.MyMindID, DailyFilename(b.Card.CapturedAt, b.Card.Title), writtenThisBatch)
		writtenThisBatch[name] = true

		content := RenderMarkdown(b.Card, nil)
		dest := filepath.Join(w.Root, name)

		if w.DryRun {
			r.CardsWritten++
			continue
		}
		if same, err := sameContent(dest, content); err == nil && same {
			r.CardsUnchanged++
			continue
		}
		if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
			return r, err
		}
		r.CardsWritten++
	}
	return r, nil
}

// resolveFilename picks a vault-relative filename for a card. If an existing
// file on disk already belongs to the same mymind_id, reuse its name (lets
// re-exports overwrite in place). Otherwise bump a collision suffix until the
// name is free.
func (w *Writer) resolveFilename(myMindID, base string, batch map[string]bool) string {
	return UniqueFilename(base, func(n string) bool {
		if batch[n] {
			return true
		}
		p := filepath.Join(w.Root, n)
		if _, err := os.Stat(p); err != nil {
			return false
		}
		// File exists on disk. If it belongs to this same mymind_id treat it
		// as the card's home, not a collision.
		if existingID := readMyMindID(p); existingID == myMindID {
			return false
		}
		return true
	})
}

func readMyMindID(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.SplitN(string(b), "\n", 20) {
		if strings.HasPrefix(line, "mymind_id: ") {
			return strings.TrimPrefix(line, "mymind_id: ")
		}
	}
	return ""
}

func sameContent(path, content string) (bool, error) {
	existing, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return string(existing) == content, nil
}

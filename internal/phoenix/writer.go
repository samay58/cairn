package phoenix

import (
	"os"
	"path/filepath"
)

func (w *Writer) Write(bundles []CardBundle) (WriteReport, error) {
	var r WriteReport
	if !w.DryRun {
		if err := os.MkdirAll(w.Root, 0o755); err != nil {
			return r, err
		}
	}
	existing := make(map[string]bool)
	for _, b := range bundles {
		name := DailyFilename(b.Card.CapturedAt, b.Card.Title)
		name = UniqueFilename(name, func(n string) bool {
			if existing[n] {
				return true
			}
			_, err := os.Stat(filepath.Join(w.Root, n))
			return err == nil
		})
		existing[name] = true
		r.CardsWritten++
		if w.DryRun {
			continue
		}
	}
	return r, nil
}

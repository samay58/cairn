package phoenix

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
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

		refs := make([]MediaRef, 0, len(b.Media))
		for _, m := range b.Media {
			rel := MediaRelPath(m.SHA256, extFromPath(m.Path))
			refs = append(refs, MediaRef{Filename: filepath.Base(m.Path), RelPath: rel})
			if w.DryRun {
				r.MediaWritten++
				continue
			}
			dest := filepath.Join(w.Root, rel)
			if existsSha(dest, m.SHA256) {
				r.MediaSkipped++
				continue
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return r, err
			}
			if err := copyFile(m.Path, dest); err != nil {
				r.Warnings = append(r.Warnings, "copy "+m.Path+": "+err.Error())
				continue
			}
			r.MediaWritten++
		}

		content := RenderMarkdown(b.Card, refs)
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

func extFromPath(p string) string {
	e := filepath.Ext(p)
	if e == "" {
		return "bin"
	}
	return strings.TrimPrefix(e, ".")
}

func existsSha(dest, wantSha string) bool {
	f, err := os.Open(dest)
	if err != nil {
		return false
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false
	}
	return hex.EncodeToString(h.Sum(nil)) == wantSha
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

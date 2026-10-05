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
	owner := w.indexByMyMindID()
	byID := make(map[string]string, len(owner))
	for name, id := range owner {
		byID[id] = name
	}
	writtenThisBatch := make(map[string]bool)
	for _, b := range bundles {
		base := DailyFilename(b.Card.CapturedAt, b.Card.Title)
		name, renameFrom := w.resolveFilename(b.Card.MyMindID, base, byID, owner, writtenThisBatch)
		writtenThisBatch[name] = true
		if renameFrom != "" && !w.DryRun {
			if err := os.Rename(filepath.Join(w.Root, renameFrom), filepath.Join(w.Root, name)); err != nil {
				return r, err
			}
			owner[name] = b.Card.MyMindID
			delete(owner, renameFrom)
			r.CardsRenamed++
		}

		refs := make([]MediaRef, 0, len(b.Media))
		for _, m := range b.Media {
			rel := MediaRelPath(m.SHA256, extFromPath(m.Path))
			ref := MediaRef{Filename: filepath.Base(m.Path), RelPath: rel}
			if w.DryRun {
				refs = append(refs, ref)
				r.MediaWritten++
				continue
			}
			dest := filepath.Join(w.Root, rel)
			if existsSha(dest, m.SHA256) {
				refs = append(refs, ref)
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
			refs = append(refs, ref)
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

// resolveFilename picks a vault-relative filename for a card. A card keeps the
// file it already owns, whatever its title is now, so links and paths stay
// stable. The one exception is a placeholder "...-untitled" name, which moves
// to the real title once MyMind has one (renameFrom is then the old name).
// A new card takes its dated slug, bumping a suffix past names other cards
// own.
func (w *Writer) resolveFilename(myMindID, base string, byID, owner map[string]string, batch map[string]bool) (name, renameFrom string) {
	taken := func(n string) bool {
		if batch[n] {
			return true
		}
		if id, ok := owner[n]; ok {
			return id != myMindID
		}
		_, err := os.Stat(filepath.Join(w.Root, n))
		return err == nil
	}
	current, ok := byID[myMindID]
	if !ok {
		return UniqueFilename(base, taken), ""
	}
	if isPlaceholderName(current) && !isPlaceholderName(base) {
		if next := UniqueFilename(base, taken); next != current {
			return next, current
		}
	}
	return current, ""
}

func isPlaceholderName(name string) bool {
	stem := strings.TrimSuffix(name, ".md")
	return len(stem) > 11 && strings.HasPrefix(stem[11:], "untitled")
}

// indexByMyMindID maps each markdown file in the vault root to the card that
// owns it, read from frontmatter.
func (w *Writer) indexByMyMindID() map[string]string {
	owner := map[string]string{}
	entries, err := os.ReadDir(w.Root)
	if err != nil {
		return owner
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if id := readMyMindID(filepath.Join(w.Root, e.Name())); id != "" {
			owner[e.Name()] = id
		}
	}
	return owner
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

package phoenix

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// RelinkReport counts what RelinkRaw did to an export folder.
type RelinkReport struct {
	Linked        int      `json:"linked"`
	AlreadyLinked int      `json:"already_linked"`
	Missing       []string `json:"missing"`
}

// RelinkRaw replaces each attachment in a MyMind export folder with a relative
// symlink to its byte-identical copy under root's _media tree. A fresh export
// drop writes every attachment again as a real file; after a sync the raw
// folder holds only cards.csv and links, so the vault keeps one copy of each
// binary. Files with no canonical copy are left in place and reported.
func RelinkRaw(rawDir, root string) (RelinkReport, error) {
	var r RelinkReport
	err := filepath.WalkDir(rawDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != rawDir && d.Name()[0] == '.' {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Type()&fs.ModeSymlink != 0 {
			r.AlreadyLinked++
			return nil
		}
		if !d.Type().IsRegular() || d.Name() == "cards.csv" || d.Name()[0] == '.' {
			return nil
		}
		sha, err := fileSha(path)
		if err != nil {
			return err
		}
		canonical := filepath.Join(root, MediaRelPath(sha, extFromPath(path)))
		if !existsSha(canonical, sha) {
			r.Missing = append(r.Missing, path)
			return nil
		}
		rel, err := filepath.Rel(filepath.Dir(path), canonical)
		if err != nil {
			return err
		}
		// Link beside the file, then rename over it, so the attachment is
		// never absent if the process stops halfway.
		tmp := path + ".cairn-link"
		_ = os.Remove(tmp)
		if err := os.Symlink(rel, tmp); err != nil {
			return err
		}
		if err := os.Rename(tmp, path); err != nil {
			_ = os.Remove(tmp)
			return err
		}
		r.Linked++
		return nil
	})
	return r, err
}

func fileSha(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

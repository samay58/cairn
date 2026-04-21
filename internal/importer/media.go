package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type MediaItem struct {
	Path   string
	SHA256 string
	Mime   string
}

func ScanMedia(dir string) ([]MediaItem, error) {
	var out []MediaItem
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		buf := make([]byte, 512)
		n, _ := f.Read(buf)
		mime := http.DetectContentType(buf[:n])
		if _, err := f.Seek(0, 0); err != nil {
			return err
		}
		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return err
		}
		out = append(out, MediaItem{
			Path:   path,
			SHA256: hex.EncodeToString(h.Sum(nil)),
			Mime:   mime,
		})
		return nil
	})
	return out, err
}

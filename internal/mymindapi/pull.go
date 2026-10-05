package mymindapi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samay58/cairn/internal/importer"
)

// Pull is one library snapshot read through the API.
type Pull struct {
	Snapshot     importer.Snapshot
	BlobsFetched int
	Quota        Quota
	Credits      int // spent by this pull
}

// PullLibrary lists every object, converts it to a card, and downloads any
// attachment not already in mediaDir (named <card id>.<ext>, the export's
// layout), so the importer indexes API and export media the same way.
func PullLibrary(ctx context.Context, c *Client, mediaDir string) (Pull, error) {
	var p Pull
	objs, q, err := c.ListObjects(ctx)
	p.Quota, p.Credits = q, q.Cost
	if err != nil {
		return p, err
	}
	have, err := mediaStems(mediaDir)
	if err != nil {
		return p, err
	}
	p.Snapshot.RowsRead = len(objs)
	for _, o := range objs {
		card, warn, err := o.ToCard()
		if err != nil {
			p.Snapshot.SkippedRows++
			p.Snapshot.Warnings = append(p.Snapshot.Warnings, err.Error())
			continue
		}
		if warn != "" {
			p.Snapshot.Warnings = append(p.Snapshot.Warnings, card.MyMindID+": "+warn)
		}
		p.Snapshot.Cards = append(p.Snapshot.Cards, card)
		if o.Blob == nil || have[card.MyMindID] {
			continue
		}
		data, mime, bq, err := c.Blob(ctx, o.ID)
		p.Credits += bq.Cost
		if bq.SustainedLimit > 0 {
			p.Quota = bq
		}
		if err != nil {
			p.Snapshot.Warnings = append(p.Snapshot.Warnings, fmt.Sprintf("attachment for %s: %v", card.MyMindID, err))
			continue
		}
		// The object's declared type names the file; the download's
		// Content-Type is a fallback.
		ext := BlobExtension(o.Blob.Type)
		if ext == "bin" {
			ext = BlobExtension(mime)
		}
		dest := filepath.Join(mediaDir, card.MyMindID+"."+ext)
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return p, err
		}
		p.BlobsFetched++
	}
	return p, nil
}

// mediaStems lists the card ids that already have an attachment (file or
// link) in dir.
func mediaStems(dir string) (map[string]bool, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	stems := make(map[string]bool, len(entries))
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || name == "cards.csv" || strings.HasPrefix(name, ".") {
			continue
		}
		stems[strings.TrimSuffix(name, filepath.Ext(name))] = true
	}
	return stems, nil
}

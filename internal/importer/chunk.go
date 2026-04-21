package importer

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type ChunkItem struct {
	Text        string
	StartOffset int
	EndOffset   int
	Checksum    string
}

const (
	targetMinWords = 200
	targetMaxWords = 600
)

// Chunk splits a card body into paragraphs, merging short adjacent ones until
// they hit the Phase 1 target (roughly 200 to 600 words) so Phase 2 embeddings
// see semantic units rather than arbitrary slices.
func Chunk(body string) []ChunkItem {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	paras := splitParagraphs(body)
	if len(paras) == 0 {
		return nil
	}
	var chunks []ChunkItem
	var buf strings.Builder
	bufWords := 0
	start := paras[0].start
	for _, p := range paras {
		pWords := len(strings.Fields(p.text))
		if bufWords+pWords > targetMaxWords && bufWords >= targetMinWords {
			chunks = append(chunks, makeChunk(buf.String(), start, p.start))
			buf.Reset()
			bufWords = 0
			start = p.start
		}
		if buf.Len() > 0 {
			buf.WriteString("\n\n")
		}
		buf.WriteString(p.text)
		bufWords += pWords
	}
	if buf.Len() > 0 {
		chunks = append(chunks, makeChunk(buf.String(), start, start+buf.Len()))
	}
	return chunks
}

type paraSpan struct {
	text  string
	start int
}

func splitParagraphs(body string) []paraSpan {
	var out []paraSpan
	pos := 0
	for _, chunk := range strings.Split(body, "\n\n") {
		if trimmed := strings.TrimSpace(chunk); trimmed != "" {
			out = append(out, paraSpan{text: trimmed, start: pos})
		}
		pos += len(chunk) + 2
	}
	return out
}

func makeChunk(text string, start, end int) ChunkItem {
	h := sha256.Sum256([]byte(text))
	return ChunkItem{
		Text:        text,
		StartOffset: start,
		EndOffset:   end,
		Checksum:    hex.EncodeToString(h[:]),
	}
}

package render

import (
	"bytes"
	"encoding/json"
	"fmt"
)

func CardListJSON(matches []Match) string {
	return JSON(CardListItems(matches))
}

func CardListJSONL(matches []Match) string {
	return JSONL(CardListItems(matches))
}

func JSON(v any) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		panic(fmt.Sprintf("render: encode json: %v", err))
	}
	return buf.String()
}

func JSONL[T any](items []T) string {
	var buf bytes.Buffer
	for _, item := range items {
		line, err := json.Marshal(item)
		if err != nil {
			panic(fmt.Sprintf("render: encode jsonl: %v", err))
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}

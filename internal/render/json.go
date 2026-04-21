package render

import (
	"bytes"
	"encoding/json"
)

func CardListJSON(matches []Match) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(matches); err != nil {
		return ""
	}
	return buf.String()
}

func CardListJSONL(matches []Match) string {
	var buf bytes.Buffer
	for _, m := range matches {
		line, _ := json.Marshal(m)
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.String()
}

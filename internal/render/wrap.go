package render

import (
	"strings"
	"unicode/utf8"
)

const DefaultWidth = 80

func WrapLines(prefix, text string, width int) []string {
	if width <= 0 {
		width = DefaultWidth
	}

	available := width - utf8.RuneCountInString(prefix)
	if available <= 0 {
		return []string{prefix + text}
	}

	paragraphs := strings.Split(text, "\n")
	lines := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, prefix)
			continue
		}

		line := words[0]
		lineLen := utf8.RuneCountInString(line)
		for _, word := range words[1:] {
			wordLen := utf8.RuneCountInString(word)
			if lineLen+1+wordLen > available {
				lines = append(lines, prefix+line)
				line = word
				lineLen = wordLen
				continue
			}
			line += " " + word
			lineLen += 1 + wordLen
		}
		lines = append(lines, prefix+line)
	}
	return lines
}

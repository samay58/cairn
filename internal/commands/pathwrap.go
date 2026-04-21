package commands

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/samay58/cairn/internal/render"
)

func writeDatabaseLocation(out io.Writer, path string) error {
	line := fmt.Sprintf("Database at %s.", path)
	if utf8.RuneCountInString(line) <= render.DefaultWidth {
		_, err := fmt.Fprintln(out, line)
		return err
	}
	if _, err := fmt.Fprintln(out, "Database at"); err != nil {
		return err
	}
	for _, line := range wrapFilesystemPath("  ", path, render.DefaultWidth) {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func writeStorageLocation(out io.Writer, path, size, mediaCache string) error {
	inline := fmt.Sprintf("storage   %s (%s) · media cache %s", path, size, mediaCache)
	if utf8.RuneCountInString(inline) <= render.DefaultWidth {
		_, err := fmt.Fprintln(out, inline)
		return err
	}
	if _, err := fmt.Fprintf(out, "storage   %s · media cache %s\n", size, mediaCache); err != nil {
		return err
	}
	for _, line := range wrapFilesystemPath("          ", path, render.DefaultWidth) {
		if _, err := fmt.Fprintln(out, line); err != nil {
			return err
		}
	}
	return nil
}

func wrapFilesystemPath(prefix, path string, width int) []string {
	if width <= 0 {
		width = render.DefaultWidth
	}
	if utf8.RuneCountInString(prefix+path) <= width {
		return []string{prefix + path}
	}

	separator := string(filepath.Separator)
	trimmed := strings.TrimPrefix(filepath.Clean(path), separator)
	parts := strings.Split(trimmed, separator)

	lines := make([]string, 0, len(parts))
	current := ""
	if strings.HasPrefix(path, separator) {
		current = separator
	}
	available := width - utf8.RuneCountInString(prefix)
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		candidate := part
		switch current {
		case "":
			candidate = part
		case separator:
			candidate = current + part
		default:
			candidate = current + separator + part
		}
		if utf8.RuneCountInString(candidate) > available && current != "" && current != separator {
			lines = append(lines, prefix+current)
			current = part
			continue
		}
		current = candidate
	}
	if current != "" {
		lines = append(lines, prefix+current)
	}
	if len(lines) == 0 {
		return []string{prefix + path}
	}
	return lines
}

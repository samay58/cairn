package phoenix

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const slugMaxLen = 60

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

func Slug(title string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	ascii, _, _ := transform.String(t, title)
	s := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(ascii), "-"), "-")
	if s == "" {
		return "untitled"
	}
	if len(s) > slugMaxLen {
		s = strings.TrimRight(s[:slugMaxLen], "-")
	}
	return s
}

func DailyFilename(capturedAt time.Time, title string) string {
	return capturedAt.UTC().Format("2006-01-02") + "-" + Slug(title) + ".md"
}

func UniqueFilename(base string, exists func(string) bool) string {
	if !exists(base) {
		return base
	}
	stem := strings.TrimSuffix(base, ".md")
	for i := 2; ; i++ {
		cand := fmt.Sprintf("%s-%d.md", stem, i)
		if !exists(cand) {
			return cand
		}
	}
}

func MediaRelPath(sha string, ext string) string {
	if len(sha) < 4 {
		return fmt.Sprintf("_media/%s.%s", sha, ext)
	}
	return fmt.Sprintf("_media/%s/%s/%s.%s", sha[:2], sha[2:4], sha, ext)
}

// RelMediaLink returns the card-relative link for a stored media path. Cards
// live at the vault root alongside `_media/`, so the link equals the stored
// path.
func RelMediaLink(relPath string) string {
	return relPath
}

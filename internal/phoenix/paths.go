package phoenix

import (
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

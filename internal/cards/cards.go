package cards

import (
	"fmt"
	"time"
)

type Kind string

const (
	KindArticle Kind = "article"
	KindImage   Kind = "image"
	KindQuote   Kind = "quote"
	KindNote    Kind = "note"
)

func (k Kind) Letter() string {
	switch k {
	case KindArticle:
		return "a"
	case KindImage:
		return "i"
	case KindQuote:
		return "q"
	case KindNote:
		return "n"
	}
	return "?"
}

func KindFromString(s string) (Kind, error) {
	switch s {
	case "article":
		return KindArticle, nil
	case "image":
		return KindImage, nil
	case "quote":
		return KindQuote, nil
	case "note":
		return KindNote, nil
	}
	return "", fmt.Errorf("unknown kind %q", s)
}

type Card struct {
	ID         string    `json:"id"`
	MyMindID   string    `json:"mymind_id"`
	Kind       Kind      `json:"kind"`
	Title      string    `json:"title"`
	URL        string    `json:"url,omitempty"`
	Body       string    `json:"body,omitempty"`
	Excerpt    string    `json:"excerpt,omitempty"`
	Source     string    `json:"source,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	CapturedAt time.Time `json:"captured_at"`
}

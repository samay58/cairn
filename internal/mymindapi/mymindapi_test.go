package mymindapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTokenIsVerifiableHS256(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	c := &Client{Creds: Credentials{KeyID: "kid1", Secret: secret}, Now: func() time.Time { return time.Unix(1700000000, 0) }}
	parts := strings.Split(c.token("GET", "/objects"), ".")
	if len(parts) != 3 {
		t.Fatalf("token has %d parts", len(parts))
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	if base64.RawURLEncoding.EncodeToString(mac.Sum(nil)) != parts[2] {
		t.Fatal("signature does not verify")
	}
	var head map[string]string
	var claims map[string]any
	decode(t, parts[0], &head)
	decode(t, parts[1], &claims)
	if head["alg"] != "HS256" || head["kid"] != "kid1" {
		t.Errorf("header = %v", head)
	}
	if claims["method"] != "GET" || claims["path"] != "/objects" || claims["exp"].(float64)-claims["iat"].(float64) != 300 {
		t.Errorf("claims = %v", claims)
	}
}

func decode(t *testing.T, seg string, v any) {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(seg)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatal(err)
	}
}

func TestParseQuota(t *testing.T) {
	h := http.Header{}
	h.Set("RateLimit", `"burst";r=9967;t=261, "sustained";r=99967;t=2591961`)
	h.Set("RateLimit-Policy", `"burst";q=10000;w=300, "sustained";q=100000;w=2592000`)
	h.Set("RateLimit-Cost", "1")
	q := parseQuota(h)
	if q.Cost != 1 || q.SustainedRemaining != 99967 || q.SustainedLimit != 100000 || q.BurstResetSeconds != 261 {
		t.Errorf("quota = %+v", q)
	}
}

func TestToCard(t *testing.T) {
	var o Object
	raw := `{
		"id": "00000000MDE0O4abcdefgh",
		"entityType": "XPost",
		"mainEntity": {"title": "Fallback title", "publication": {"name": "X"}},
		"summary": "[[Anthropic]] met [[Vatican|church]] leaders.",
		"notes": [{"content": {"type": "application/prose+json", "body": {"type": "doc", "content": [
			{"type": "paragraph", "content": [{"type": "text", "text": "first"}]},
			{"type": "paragraph"},
			{"type": "paragraph", "content": [{"type": "text", "text": "second"}]}]}}}],
		"tags": [{"name": "AI"}, {"name": "AI"}, {"name": " ethics "}],
		"source": {"url": "https://example.com/a?utm_source=x&id=7"},
		"created": "2026-10-05T17:16:27.231183Z"
	}`
	if err := json.Unmarshal([]byte(raw), &o); err != nil {
		t.Fatal(err)
	}
	c, warn, err := o.ToCard()
	if err != nil || warn != "" {
		t.Fatalf("ToCard: %v %q", err, warn)
	}
	want := "Anthropic met church leaders.\n\nNote: first\n\nsecond"
	switch {
	case c.MyMindID != "MDE0O4abcdefgh" || c.ID != c.MyMindID:
		t.Errorf("id = %q", c.MyMindID)
	case c.Title != "Fallback title":
		t.Errorf("title = %q", c.Title)
	case c.Body != want:
		t.Errorf("body = %q, want %q", c.Body, want)
	case c.URL != "https://example.com/a?id=7":
		t.Errorf("url = %q", c.URL)
	case strings.Join(c.Tags, "|") != "AI|ethics":
		t.Errorf("tags = %v", c.Tags)
	case c.Source != "X" || c.CapturedAt.Nanosecond() != 0:
		t.Errorf("source %q, captured %v", c.Source, c.CapturedAt)
	}
}

func TestToCardUsesOwnContentForNotes(t *testing.T) {
	var o Object
	raw := `{"id": "00000000MDE0O0N5xTL9Hq", "entityType": "Note", "summary": "ignored",
		"content": {"type": "text/plain", "body": "hello"}, "created": "2025-02-20T09:07:08.245913Z"}`
	if err := json.Unmarshal([]byte(raw), &o); err != nil {
		t.Fatal(err)
	}
	c, _, err := o.ToCard()
	if err != nil || c.Body != "hello" || c.Kind != "note" {
		t.Fatalf("card = %+v, err %v", c, err)
	}
}

func TestBlobExtension(t *testing.T) {
	for mime, want := range map[string]string{"application/pdf": "pdf", "image/jpeg; q=1": "jpeg", "application/octet-stream": "bin"} {
		if got := BlobExtension(mime); got != want {
			t.Errorf("BlobExtension(%q) = %q, want %q", mime, got, want)
		}
	}
}

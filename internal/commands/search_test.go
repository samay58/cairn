package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestSearchOAuthPlain(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "oauth"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_oauth.txt", out.String())
}

func TestSearchOAuthJSON(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "oauth", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	var got []struct {
		Handle   int    `json:"handle"`
		WhyShown string `json:"why_shown"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal search json: %v", err)
	}
	for i, item := range got {
		if item.Handle != i+1 {
			t.Fatalf("handle[%d] = %d, want %d", i, item.Handle, i+1)
		}
		if item.WhyShown == "" {
			t.Fatalf("why_shown[%d] is empty", i)
		}
	}
	golden.Assert(t, "search_oauth.json", out.String())
}

func TestSearchWithLimit(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "oauth", "--limit", "2"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_oauth_limit2.txt", out.String())
}

func TestSearchOAuthJSONL(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "oauth", "--jsonl"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_oauth.jsonl", out.String())
}

func TestSearchEmpty(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"search", "zzz-empty"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "search_empty.txt", out.String())
}

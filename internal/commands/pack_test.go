package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestPackClaude(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"pack", "oauth device flow", "--for", "claude"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "pack_claude.txt", out.String())
}

func TestPackJSON(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"pack", "oauth device flow", "--for", "json"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "pack_json.txt", out.String())
}

func TestPackHonorsLimit(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"pack", "oauth device flow", "--limit", "1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `cards="1"`) {
		t.Fatalf("expected cards=1 in pack output, got:\n%s", out.String())
	}
	if strings.Contains(out.String(), "c_0011") {
		t.Fatalf("expected limited pack output to omit later cards, got:\n%s", out.String())
	}
}

func TestPackEscapesXMLQuery(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"pack", "a & b"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `query="a &amp; b"`) {
		t.Fatalf("expected escaped xml query, got:\n%s", out.String())
	}
}

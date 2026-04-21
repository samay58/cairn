package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/samay58/cairn/internal/golden"
	"github.com/samay58/cairn/internal/source"
)

func TestOpenByHandle(t *testing.T) {
	t.Setenv("CAIRN_HOME", t.TempDir())
	t.Setenv("CAIRN_DRY_OPEN", "1")

	root := NewRootWithSource(source.NewFixtureSource())
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"open", "@1"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "open_at1.txt", out.String())
}

func TestOpenReturnsErrorWhenBrowserLaunchFails(t *testing.T) {
	t.Setenv("PATH", "")

	root := NewRootWithSource(source.NewFixtureSource())
	root.SetArgs([]string{"open", "@1"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected open error")
	}
	if !strings.Contains(err.Error(), "open card: launch browser:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

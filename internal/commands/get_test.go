package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestGetByHandle(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"get", "@2"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "get_at2.txt", out.String())
}

func TestGetUnknownHandle(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"get", "@99"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected error for unknown handle")
	}
	golden.Assert(t, "get_err_unknown.txt", err.Error()+"\n")
}

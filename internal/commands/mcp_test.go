package commands

import (
	"bytes"
	"testing"

	"github.com/samay58/cairn/internal/golden"
)

func TestMCPStart(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "start"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_start.txt", out.String())
}

func TestMCPInstallClaudeCode(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "install", "claude-code"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_install_claude_code.txt", out.String())
}

func TestMCPInstallClaudeDesktop(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "install", "claude-desktop"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_install_claude_desktop.txt", out.String())
}

func TestMCPAudit(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "audit"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_audit.txt", out.String())
}

func TestMCPInstallManual(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "install", "manual"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_install_manual.txt", out.String())
}

func TestMCPPermissions(t *testing.T) {
	root := NewRoot()
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"mcp", "permissions"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	golden.Assert(t, "mcp_permissions.txt", out.String())
}

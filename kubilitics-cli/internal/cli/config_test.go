package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestConfigCommandRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := NewRootCommand()
	out := &bytes.Buffer{}
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"config", "set", "tui.refresh_interval", "3s"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set failed: %v", err)
	}

	root = NewRootCommand()
	out.Reset()
	root.SetOut(out)
	root.SetErr(out)
	root.SetArgs([]string{"config", "get", "tui.refresh_interval"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config get failed: %v", err)
	}
	if strings.TrimSpace(out.String()) != "3s" {
		t.Fatalf("unexpected config get output: %q", out.String())
	}
}

func TestConfigResetRequiresYes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	root := NewRootCommand()
	root.SetArgs([]string{"config", "reset"})
	if err := root.Execute(); err == nil {
		t.Fatal("expected reset without --yes to fail")
	}
}

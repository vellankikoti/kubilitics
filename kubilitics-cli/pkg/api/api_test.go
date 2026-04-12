package api

import (
	"strings"
	"testing"
)

func TestExecuteVersion(t *testing.T) {
	c := NewKCLI(Config{})
	out, err := c.Execute("version")
	if err != nil {
		t.Fatalf("execute version error: %v", err)
	}
	if !strings.Contains(strings.ToLower(out), "kcli") {
		t.Fatalf("unexpected version output: %q", out)
	}
}

func TestExecuteStreamVersion(t *testing.T) {
	c := NewKCLI(Config{})
	ch, err := c.ExecuteStream("kcli version")
	if err != nil {
		t.Fatalf("execute stream error: %v", err)
	}
	seen := false
	for msg := range ch {
		if msg.Done {
			if msg.Err != nil {
				t.Fatalf("stream done with error: %v", msg.Err)
			}
			continue
		}
		if strings.Contains(strings.ToLower(msg.Data), "kcli") {
			seen = true
		}
	}
	if !seen {
		t.Fatal("expected streamed kcli output")
	}
}

func TestParseCommand(t *testing.T) {
	if got := parseCommand("kcli get pods"); len(got) != 2 || got[0] != "get" || got[1] != "pods" {
		t.Fatalf("unexpected parse: %+v", got)
	}
	if got := parseCommand("get pods"); len(got) != 2 || got[0] != "get" || got[1] != "pods" {
		t.Fatalf("unexpected parse: %+v", got)
	}
}

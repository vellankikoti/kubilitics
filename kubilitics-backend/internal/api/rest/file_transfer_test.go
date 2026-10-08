package rest

import (
	"fmt"
	"strings"
	"testing"
)

// This file had zero test coverage before this change despite containing
// path-sanitization (security-relevant) and output-parsing logic, plus the
// new truncation cap added to fix a reported bug: ListContainerFiles
// (PVC file browser's "ls") returned a bare, unbounded JSON array that the
// frontend rendered un-virtualized — a directory with thousands of entries
// froze the UI.

func TestSanitizeContainerPath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"empty defaults to root", "", "/", false},
		{"already absolute", "/data", "/data", false},
		{"relative gets rooted", "data", "/data", false},
		{"cleans trailing slash", "/data/", "/data", false},
		{"cleans double slashes", "/data//logs", "/data/logs", false},
		// path.Clean on an ALREADY-ABSOLUTE path can never escape above "/" —
		// Go's Clean resolves ".." against real preceding components and
		// drops any excess ".." at the root, so these are safely normalized,
		// not traversal. The real danger path.Clean can't protect against on
		// its own is a RELATIVE input where the leading slash is added
		// *after* Clean runs (see "rejects leading traversal" below) — that's
		// what the subsequent "contains .." check actually catches.
		{"absolute parent-ref resolves safely, not an error", "/data/../etc/passwd", "/etc/passwd", false},
		{"absolute excess parent-refs resolve safely, not an error", "/data/../../etc", "/etc", false},
		{"rejects leading traversal", "../etc/passwd", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeContainerPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error for input %q, got none (result: %q)", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tt.input, err)
			}
			if got != tt.want {
				t.Fatalf("input %q: want %q, got %q", tt.input, tt.want, got)
			}
		})
	}
}

func TestParseLsOutput_GNULongISO(t *testing.T) {
	output := strings.Join([]string{
		"total 16",
		"drwxr-xr-x 2 root root 4096 2024-01-15 10:30 subdir",
		"-rw-r--r-- 1 root root 1234 2024-01-15 10:31 config.yaml",
		"lrwxrwxrwx 1 root root    7 2024-01-15 10:32 link -> target",
	}, "\n")

	entries := parseLsOutput(output)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %+v", len(entries), entries)
	}

	byName := map[string]FileEntry{}
	for _, e := range entries {
		byName[e.Name] = e
	}

	if byName["subdir"].Type != "dir" {
		t.Errorf("expected subdir to be type dir, got %q", byName["subdir"].Type)
	}
	if byName["config.yaml"].Type != "file" || byName["config.yaml"].Size != 1234 {
		t.Errorf("expected config.yaml to be a 1234-byte file, got %+v", byName["config.yaml"])
	}
	if byName["link"].Type != "link" {
		t.Errorf("expected link to be type link, got %q", byName["link"].Type)
	}
}

func TestParseLsOutput_SkipsDotAndDotDot(t *testing.T) {
	output := strings.Join([]string{
		"total 8",
		"drwxr-xr-x 3 root root 4096 2024-01-15 10:30 .",
		"drwxr-xr-x 3 root root 4096 2024-01-15 10:30 ..",
		"-rw-r--r-- 1 root root    0 2024-01-15 10:30 empty.txt",
	}, "\n")

	entries := parseLsOutput(output)
	if len(entries) != 1 {
		t.Fatalf("expected '.' and '..' to be filtered out, got %d entries: %+v", len(entries), entries)
	}
	if entries[0].Name != "empty.txt" {
		t.Errorf("expected empty.txt, got %q", entries[0].Name)
	}
}

func TestParseLsOutput_EmptyDirectory(t *testing.T) {
	entries := parseLsOutput("total 0")
	if len(entries) != 0 {
		t.Fatalf("expected no entries for an empty directory, got %d", len(entries))
	}
}

// TestCapEntries is the regression test for the unbounded-listing bug: a
// directory with more than maxLsEntries files must come back truncated,
// not as one giant array.
func TestCapEntries(t *testing.T) {
	t.Run("under the cap: unchanged, not truncated", func(t *testing.T) {
		entries := make([]FileEntry, 5)
		got, truncated := capEntries(entries, maxLsEntries)
		if truncated {
			t.Error("expected truncated=false when under the cap")
		}
		if len(got) != 5 {
			t.Errorf("expected 5 entries, got %d", len(got))
		}
	})

	t.Run("over the cap: bounded and flagged", func(t *testing.T) {
		entries := make([]FileEntry, maxLsEntries+500)
		for i := range entries {
			entries[i] = FileEntry{Name: fmt.Sprintf("file-%d", i)}
		}
		got, truncated := capEntries(entries, maxLsEntries)
		if !truncated {
			t.Error("expected truncated=true when over the cap")
		}
		if len(got) != maxLsEntries {
			t.Fatalf("expected exactly %d entries, got %d — this is what prevents the frontend from freezing on a huge directory", maxLsEntries, len(got))
		}
	})

	t.Run("exactly at the cap: not truncated", func(t *testing.T) {
		entries := make([]FileEntry, maxLsEntries)
		got, truncated := capEntries(entries, maxLsEntries)
		if truncated {
			t.Error("expected truncated=false when exactly at the cap")
		}
		if len(got) != maxLsEntries {
			t.Errorf("expected %d entries, got %d", maxLsEntries, len(got))
		}
	})
}

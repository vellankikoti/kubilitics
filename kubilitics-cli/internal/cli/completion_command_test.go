package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestCompletionScriptsForAllShells(t *testing.T) {
	shells := []string{"bash", "zsh", "fish", "powershell"}
	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			root := NewRootCommand()
			buf := &bytes.Buffer{}
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs([]string{"completion", sh})
			if err := root.Execute(); err != nil {
				t.Fatalf("completion %s failed: %v", sh, err)
			}
			if buf.Len() == 0 {
				t.Fatalf("completion %s returned empty script", sh)
			}
			if !strings.Contains(strings.ToLower(buf.String()), "kcli") {
				t.Fatalf("completion %s output missing command name", sh)
			}
		})
	}
}

func TestCachedKubectlHotPathLatency(t *testing.T) {
	a := &app{cache: initCache(), completionTimeout: 200 * time.Millisecond}
	a.cache["hot"] = cacheEntry{value: "pod/a\npod/b", expires: time.Now().Add(30 * time.Second)}

	start := time.Now()
	iterations := 5000
	for i := 0; i < iterations; i++ {
		_, err := a.cachedKubectl("hot", []string{"get", "pods", "-o", "name"}, 30*time.Second)
		if err != nil {
			t.Fatalf("cachedKubectl returned error: %v", err)
		}
	}
	avg := time.Since(start) / time.Duration(iterations)
	if avg > 50*time.Millisecond {
		t.Fatalf("cachedKubectl hot path too slow: avg=%s", avg)
	}
}

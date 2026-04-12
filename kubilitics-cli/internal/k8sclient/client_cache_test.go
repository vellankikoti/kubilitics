package k8sclient

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func writeConfigFile(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	return path
}

func TestLoadRawConfigCacheTTL(t *testing.T) {
	origTTL := rawConfigCacheTTL
	rawConfigCacheTTL = 40 * time.Millisecond
	defer func() { rawConfigCacheTTL = origTTL }()

	rawConfigCacheMu.Lock()
	rawConfigCache = map[string]rawConfigCacheEntry{}
	rawConfigCacheMu.Unlock()

	path := writeConfigFile(t, `apiVersion: v1
kind: Config
current-context: c1
contexts:
- name: c1
  context:
    cluster: cl
    user: u
clusters:
- name: cl
  cluster:
    server: https://127.0.0.1:6443
users:
- name: u
  user:
    token: a
`)

	ctxs1, err := ListContexts(path)
	if err != nil {
		t.Fatalf("ListContexts(1): %v", err)
	}
	if !reflect.DeepEqual(ctxs1, []string{"c1"}) {
		t.Fatalf("unexpected contexts: %v", ctxs1)
	}

	if err := os.WriteFile(path, []byte(`apiVersion: v1
kind: Config
current-context: c2
contexts:
- name: c2
  context:
    cluster: cl
    user: u
clusters:
- name: cl
  cluster:
    server: https://127.0.0.1:6443
users:
- name: u
  user:
    token: b
`), 0o600); err != nil {
		t.Fatalf("rewrite config: %v", err)
	}

	ctxsCached, err := ListContexts(path)
	if err != nil {
		t.Fatalf("ListContexts(cached): %v", err)
	}
	if !reflect.DeepEqual(ctxsCached, []string{"c1"}) {
		t.Fatalf("expected cached contexts [c1], got %v", ctxsCached)
	}

	time.Sleep(55 * time.Millisecond)
	ctxs2, err := ListContexts(path)
	if err != nil {
		t.Fatalf("ListContexts(2): %v", err)
	}
	if !reflect.DeepEqual(ctxs2, []string{"c2"}) {
		t.Fatalf("expected refreshed contexts [c2], got %v", ctxs2)
	}
}

func TestBundleCacheTTL(t *testing.T) {
	origTTL := bundleCacheTTL
	bundleCacheTTL = 40 * time.Millisecond
	defer func() { bundleCacheTTL = origTTL }()

	bundleCacheMu.Lock()
	bundleCache = map[string]bundleCacheEntry{}
	bundleCacheMu.Unlock()

	path := writeConfigFile(t, `apiVersion: v1
kind: Config
current-context: c1
contexts:
- name: c1
  context:
    cluster: cl
    user: u
clusters:
- name: cl
  cluster:
    server: https://127.0.0.1:6443
users:
- name: u
  user:
    token: a
`)

	b1, err := NewBundle(path, "c1")
	if err != nil {
		t.Fatalf("NewBundle(1): %v", err)
	}
	b2, err := NewBundle(path, "c1")
	if err != nil {
		t.Fatalf("NewBundle(2): %v", err)
	}
	if b1 != b2 {
		t.Fatal("expected cached bundle pointer to be reused within TTL")
	}

	time.Sleep(55 * time.Millisecond)
	b3, err := NewBundle(path, "c1")
	if err != nil {
		t.Fatalf("NewBundle(3): %v", err)
	}
	if b3 == b2 {
		t.Fatal("expected cache refresh after TTL")
	}
}

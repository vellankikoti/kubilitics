package k8sclient

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"k8s.io/client-go/rest"
)

func writeKubeconfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	return path
}

func TestListAndCurrentContext(t *testing.T) {
	kubeconfig := `apiVersion: v1
kind: Config
current-context: prod
contexts:
- name: prod
  context:
    cluster: c1
    user: u1
- name: dev
  context:
    cluster: c1
    user: u2
clusters:
- name: c1
  cluster:
    server: https://127.0.0.1:6443
users:
- name: u1
  user:
    token: abc
- name: u2
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: aws
      args: ["eks", "get-token"]
`
	path := writeKubeconfig(t, kubeconfig)
	ctxs, err := ListContexts(path)
	if err != nil {
		t.Fatalf("ListContexts failed: %v", err)
	}
	want := []string{"dev", "prod"}
	if !reflect.DeepEqual(ctxs, want) {
		t.Fatalf("contexts mismatch: got %v want %v", ctxs, want)
	}
	current, err := CurrentContext(path)
	if err != nil {
		t.Fatalf("CurrentContext failed: %v", err)
	}
	if current != "prod" {
		t.Fatalf("current context = %q want prod", current)
	}
}

func TestDetectAuthMethods(t *testing.T) {
	kubeconfig := `apiVersion: v1
kind: Config
current-context: prod
contexts:
- name: prod
  context:
    cluster: c1
    user: u1
- name: dev
  context:
    cluster: c1
    user: u2
clusters:
- name: c1
  cluster:
    server: https://127.0.0.1:6443
users:
- name: u1
  user:
    token: abc
    client-certificate-data: "Y2VydA=="
- name: u2
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: gcloud
    auth-provider:
      name: oidc
`
	path := writeKubeconfig(t, kubeconfig)
	raw, err := loadRawConfig(path)
	if err != nil {
		t.Fatalf("loadRawConfig failed: %v", err)
	}
	got := DetectAuthMethods(raw, "prod")
	want := []string{"token", "client-cert"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("auth methods mismatch for prod: got %v want %v", got, want)
	}
	got = DetectAuthMethods(raw, "dev")
	want = []string{"exec", "auth-provider:oidc"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("auth methods mismatch for dev: got %v want %v", got, want)
	}
}

func minimalKubeconfig() string {
	return `apiVersion: v1
kind: Config
current-context: prod
contexts:
- name: prod
  context:
    cluster: c1
    user: u1
clusters:
- name: c1
  cluster:
    server: https://127.0.0.1:6443
users:
- name: u1
  user:
    token: abc
`
}

// TestNewBundleForTUI_SuppressesWarningsWithoutAffectingPlainBundle pins the
// fix for client-go's default warning handler corrupting the Bubble Tea
// TUI's alt-screen (deprecation warnings like "Warning: v1 Endpoints is
// deprecated..." were written straight to stderr, bypassing the TUI's
// managed rendering and making the header/footer appear to vanish). The fix
// must be scoped to the TUI's own Bundle via rest.Config.WarningHandler,
// never via the process-global rest.SetDefaultWarningHandler — a plain,
// non-TUI bundle built in the same process must keep the default behavior.
func TestNewBundleForTUI_SuppressesWarningsWithoutAffectingPlainBundle(t *testing.T) {
	path := writeKubeconfig(t, minimalKubeconfig())

	plain, err := NewBundle(path, "prod")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}
	if plain.RestConfig.WarningHandler != nil {
		t.Errorf("plain NewBundle must not set a custom WarningHandler (would change default kcli get/show behavior), got %T", plain.RestConfig.WarningHandler)
	}

	tui, err := NewBundleForTUI(path, "prod")
	if err != nil {
		t.Fatalf("NewBundleForTUI failed: %v", err)
	}
	if _, ok := tui.RestConfig.WarningHandler.(rest.NoWarnings); !ok {
		t.Errorf("NewBundleForTUI's RestConfig.WarningHandler = %T, want rest.NoWarnings (so warnings don't corrupt the alt-screen)", tui.RestConfig.WarningHandler)
	}
}

// TestNewBundleForTUI_BypassesSharedCache guards against a TUI-built bundle
// (warnings suppressed) or a plain bundle leaking into each other through
// the shared bundleCache, which would silently change behavior for whichever
// caller happens to run second within the cache TTL.
func TestNewBundleForTUI_BypassesSharedCache(t *testing.T) {
	path := writeKubeconfig(t, minimalKubeconfig())

	if _, err := NewBundleForTUI(path, "prod"); err != nil {
		t.Fatalf("NewBundleForTUI failed: %v", err)
	}
	plain, err := NewBundle(path, "prod")
	if err != nil {
		t.Fatalf("NewBundle failed: %v", err)
	}
	if plain.RestConfig.WarningHandler != nil {
		t.Errorf("NewBundle returned a bundle with a non-default WarningHandler (%T) — looks like it was served from cache after NewBundleForTUI populated it, or a TUI bundle leaked into the shared cache", plain.RestConfig.WarningHandler)
	}
}

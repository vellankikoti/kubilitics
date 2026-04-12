package k8sclient

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
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

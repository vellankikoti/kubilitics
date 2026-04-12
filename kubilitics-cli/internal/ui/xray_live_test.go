package ui

import (
	"os"
	"testing"
)

func TestBuildXrayLiveDeployment(t *testing.T) {
	if os.Getenv("KCLI_LIVE_TEST") != "1" {
		t.Skip("set KCLI_LIVE_TEST=1 to run live cluster relationship test")
	}
	opts := Options{Context: "docker-desktop"}
	root, err := buildXray(opts, resourceSpec{Kind: "Deployment", KubectlType: "deployments", Namespaced: true}, resourceRow{Namespace: "blue-green-demo", Name: "blue-deployment"})
	if err != nil {
		t.Fatalf("buildXray deployment failed: %v", err)
	}
	if root == nil || len(root.Children) == 0 {
		t.Fatalf("expected relationship children for deployment xray")
	}
	seen := map[string]bool{}
	var walk func(nodes []*xrayNode)
	walk = func(nodes []*xrayNode) {
		for _, n := range nodes {
			seen[n.Kind] = true
			walk(n.Children)
		}
	}
	walk(root.Children)
	if !seen["ReplicaSet"] || !seen["Pod"] || !seen["Service"] {
		t.Fatalf("expected ReplicaSet/Pod/Service relationships, got: %+v", seen)
	}
}

func TestBuildXrayLiveService(t *testing.T) {
	if os.Getenv("KCLI_LIVE_TEST") != "1" {
		t.Skip("set KCLI_LIVE_TEST=1 to run live cluster relationship test")
	}
	opts := Options{Context: "docker-desktop"}
	root, err := buildXray(opts, resourceSpec{Kind: "Service", KubectlType: "services", Namespaced: true}, resourceRow{Namespace: "blue-green-demo", Name: "demo-app"})
	if err != nil {
		t.Fatalf("buildXray service failed: %v", err)
	}
	if root == nil || len(root.Children) == 0 {
		t.Fatalf("expected relationship children for service xray")
	}
	seenPod := false
	for _, c := range root.Children {
		if c.Kind == "Pod" {
			seenPod = true
			break
		}
	}
	if !seenPod {
		t.Fatalf("expected pod relationship from service xray")
	}
}

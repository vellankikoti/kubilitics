package cli

import (
	"reflect"
	"testing"
)

func TestResourceArgIndex(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want int
	}{
		{name: "simple resource", args: []string{"pods"}, want: 0},
		{name: "with namespace flag", args: []string{"-n", "default", "pods"}, want: 2},
		{name: "with all namespaces", args: []string{"-A", "pods"}, want: 1},
		{name: "only flags", args: []string{"-A", "--namespace", "x"}, want: -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resourceArgIndex(tc.args)
			if got != tc.want {
				t.Fatalf("resourceArgIndex(%v)=%d want %d", tc.args, got, tc.want)
			}
		})
	}
}

func TestExtractScopeFlags(t *testing.T) {
	args := []string{"pods", "-A", "--context", "prod", "--namespace=default", "--context-group", "blue"}
	got := extractScopeFlags(args)
	want := []string{"-A", "--context", "prod", "--namespace=default"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractScopeFlags mismatch: got %v want %v", got, want)
	}
}

func TestAwaitingFlagValue(t *testing.T) {
	cases := []struct {
		name string
		args []string
		flag string
		ok   bool
	}{
		{name: "await context value", args: []string{"pods", "--context"}, flag: "--context", ok: true},
		{name: "await namespace short", args: []string{"pods", "-n"}, flag: "--namespace", ok: true},
		{name: "inline equals", args: []string{"pods", "--context="}, flag: "--context", ok: true},
		{name: "not awaiting", args: []string{"pods", "-A"}, flag: "", ok: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			flag, ok := awaitingFlagValue(tc.args, "get")
			if ok != tc.ok || flag != tc.flag {
				t.Fatalf("awaitingFlagValue(%v)=(%q,%v) want (%q,%v)", tc.args, flag, ok, tc.flag, tc.ok)
			}
		})
	}
}

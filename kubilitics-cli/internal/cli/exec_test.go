package cli

import (
	"reflect"
	"testing"
)

func TestHasContainerFlag(t *testing.T) {
	tests := []struct {
		args []string
		want bool
	}{
		{[]string{"get", "pods"}, false},
		{[]string{"exec", "pod1", "-c", "shell"}, true},
		{[]string{"exec", "pod1", "--container", "shell"}, true},
		{[]string{"exec", "pod1", "--container=shell"}, true},
	}
	for _, tt := range tests {
		if got := hasContainerFlag(tt.args); got != tt.want {
			t.Errorf("hasContainerFlag(%v) = %v; want %v", tt.args, got, tt.want)
		}
	}
}

func TestInsertContainerFlag(t *testing.T) {
	tests := []struct {
		args      []string
		container string
		want      []string
	}{
		{
			[]string{"pod1", "ls"},
			"main",
			[]string{"pod1", "-c", "main", "ls"},
		},
		{
			[]string{"-i", "pod1"},
			"main",
			[]string{"-i", "pod1", "-c", "main"},
		},
		{
			[]string{},
			"main",
			[]string{"-c", "main"},
		},
	}
	for _, tt := range tests {
		if got := insertContainerFlag(tt.args, tt.container); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("insertContainerFlag(%v, %s) = %v; want %v", tt.args, tt.container, got, tt.want)
		}
	}
}

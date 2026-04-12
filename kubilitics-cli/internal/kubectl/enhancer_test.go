package kubectl

import (
	"testing"
)

func TestParseWithModifiers(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantVerb      string
		wantResource  string
		wantMods      []string
		wantRemaining []string
		wantErr       bool
	}{
		{
			name:         "pods with ip,node",
			args:         []string{"get", "pods", "with", "ip,node"},
			wantVerb:     "get",
			wantResource: "pods",
			wantMods:     []string{"ip", "node"},
		},
		{
			name:         "deployments with replicas",
			args:         []string{"get", "deployments", "with", "replicas"},
			wantVerb:     "get",
			wantResource: "deployments",
			wantMods:     []string{"replicas"},
		},
		{
			name:          "pods with all and namespace",
			args:          []string{"get", "pods", "with", "all", "-n", "prod"},
			wantVerb:      "get",
			wantResource:  "pods",
			wantMods:      []string{"all"},
			wantRemaining: []string{"-n", "prod"},
		},
		{
			name:    "non-get verb returns error",
			args:    []string{"delete", "pods"},
			wantErr: true,
		},
		{
			name:    "missing resource",
			args:    []string{"get"},
			wantErr: true,
		},
		{
			name:    "empty args",
			args:    []string{},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			verb, resource, mods, remaining, err := ParseWithModifiers(tc.args)
			if tc.wantErr {
				if err == nil {
					t.Errorf("ParseWithModifiers(%v) expected error, got nil", tc.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseWithModifiers(%v) unexpected error: %v", tc.args, err)
			}
			if verb != tc.wantVerb {
				t.Errorf("verb = %q, want %q", verb, tc.wantVerb)
			}
			if resource != tc.wantResource {
				t.Errorf("resource = %q, want %q", resource, tc.wantResource)
			}
			if len(mods) != len(tc.wantMods) {
				t.Errorf("modifiers = %v, want %v", mods, tc.wantMods)
			} else {
				for i := range mods {
					if mods[i] != tc.wantMods[i] {
						t.Errorf("modifiers[%d] = %q, want %q", i, mods[i], tc.wantMods[i])
					}
				}
			}
			if tc.wantRemaining != nil {
				if len(remaining) != len(tc.wantRemaining) {
					t.Errorf("remaining = %v, want %v", remaining, tc.wantRemaining)
				}
			}
		})
	}
}

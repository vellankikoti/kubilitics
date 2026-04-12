package output

import (
	"strings"
	"testing"
	"time"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		age      time.Duration
		contains string
	}{
		{"just now", 0, ""},
		{"30 seconds", 30 * time.Second, ""},
		{"5 minutes", 5 * time.Minute, "m"},
		{"2 hours", 2 * time.Hour, "h"},
		{"3 days", 72 * time.Hour, "d"},
		{"2 weeks", 14 * 24 * time.Hour, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ts := time.Now().Add(-tc.age)
			result := FormatAge(ts)
			if result == "" {
				t.Error("FormatAge() returned empty string")
			}
			if tc.contains != "" && !strings.Contains(result, tc.contains) {
				t.Errorf("FormatAge() = %q, want to contain %q", result, tc.contains)
			}
		})
	}
}

func TestFormatAge_ZeroTime(t *testing.T) {
	result := FormatAge(time.Time{})
	if result == "" {
		t.Error("FormatAge(zero) returned empty string")
	}
}

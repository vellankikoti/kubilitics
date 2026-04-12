package output

import (
	"testing"
)

func TestGetTermCaps(t *testing.T) {
	caps := GetTermCaps()
	// In a test environment, GetTermCaps may return nil or a valid capabilities object
	if caps == nil {
		t.Skip("GetTermCaps returned nil (expected in non-TTY test environment)")
	}
	w, h := caps.GetSize()
	if w < 0 || h < 0 {
		t.Errorf("GetSize() returned negative values: w=%d, h=%d", w, h)
	}
}

func TestInit(t *testing.T) {
	// Init should not panic
	caps := Init()
	if caps == nil {
		t.Skip("Init returned nil (expected in non-TTY test environment)")
	}
}

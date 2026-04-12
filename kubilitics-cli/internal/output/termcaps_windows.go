//go:build windows

package output

// startResizeListener is a no-op on Windows. Windows terminals handle
// resize events differently (via console API) and golang.org/x/term
// does not support SIGWINCH on Windows.
func startResizeListener(_ *Capabilities) {}

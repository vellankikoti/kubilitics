//go:build !windows

package output

import (
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func startResizeListener(caps *Capabilities) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			if width, height, err := term.GetSize(int(os.Stdout.Fd())); err == nil {
				caps.SetSize(width, height)
			}
		}
	}()
}

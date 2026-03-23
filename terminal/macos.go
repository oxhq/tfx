//go:build darwin

package terminal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// isTerminal checks if fd is a terminal on macOS
// Uses TIOCGETA which is available in unix package for BSD-based systems
func isTerminal(fd uintptr) bool {
	intFD, err := fdToInt(fd)
	if err != nil {
		return false
	}
	_, err = unix.IoctlGetTermios(intFD, unix.TIOCGETA)
	return err == nil
}

// enableANSI is a no-op on macOS (ANSI is natively supported)
func enableANSI() bool {
	return true
}

// listenForSignals handles SIGWINCH (resize) and SIGINT/SIGTERM (stop) on macOS
func listenForSignals(ctx context.Context, handler *SignalHandler) {
	resizeCh := make(chan os.Signal, 1)
	stopCh := make(chan os.Signal, 1)

	signal.Notify(resizeCh, syscall.SIGWINCH)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

	defer signal.Stop(resizeCh)
	defer signal.Stop(stopCh)

	for {
		select {
		case <-ctx.Done():
			if handler.onStop != nil {
				handler.onStop()
			}
			return
		case <-handler.stopCh:
			return
		case <-resizeCh:
			if handler.onResize != nil {
				handler.onResize()
			}
		case <-stopCh:
			if handler.onStop != nil {
				handler.onStop()
			}
			return
		}
	}
}

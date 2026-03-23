//go:build linux || aix

package terminal

import (
	"context"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"golang.org/x/sys/unix"
)

// Terminal ioctl constants for Linux-based systems
const (
	// Embedded Linux variants (only define if not in unix package)
	TCGETS_EMBEDDED = 0x5400 // Some embedded systems (routers, IoT)
)

// isTerminal checks if fd is a terminal on Linux-based systems
func isTerminal(fd uintptr) bool {
	intFD, err := fdToInt(fd)
	if err != nil {
		return false
	}

	// Try unix package constant first (most reliable)
	if _, err := unix.IoctlGetTermios(intFD, unix.TCGETS); err == nil {
		return true
	}

	// Embedded Linux fallback for ARM/MIPS devices
	if isEmbeddedArch() {
		if _, err := unix.IoctlGetTermios(intFD, TCGETS_EMBEDDED); err == nil {
			return true
		}
	}

	return false
}

// isEmbeddedArch checks if we're running on embedded architecture
func isEmbeddedArch() bool {
	return runtime.GOARCH == "arm" || runtime.GOARCH == "arm64" ||
		runtime.GOARCH == "mips" || runtime.GOARCH == "mipsle" ||
		runtime.GOARCH == "mips64" || runtime.GOARCH == "mips64le"
}

// enableANSI is a no-op on Linux systems (ANSI is natively supported)
func enableANSI() bool {
	return true
}

// listenForSignals handles SIGWINCH (resize) and SIGINT/SIGTERM (stop) on Unix systems
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

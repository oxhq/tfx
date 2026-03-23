//go:build unix && !darwin && !linux && !aix

package terminal

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// Terminal ioctl constants for various Unix systems
// Covers: FreeBSD, OpenBSD, NetBSD, DragonFly, Solaris, Illumos, AIX, Android, QNX, etc.
const (
	// POSIX/Linux style (AIX, Android, embedded Linux)
	TCGETS_FALLBACK = 0x5401

	// System V style (Solaris, Illumos)
	TCGETA_FALLBACK     = 0x5405 // Modern Solaris
	TCGETA_OLD_FALLBACK = 0x5403 // Older Solaris variants

	// Embedded systems (routers, IoT devices)
	TCGETS_EMBEDDED = 0x5400

	// QNX real-time OS (automotive, medical, industrial)
	QNX_TCGETA = 0x540A
)

// isTerminal checks if fd is a terminal on various Unix systems
// Covers all Unix systems not handled by macos.go, linux.go, or windows.go
func isTerminal(fd uintptr) bool {
	// Try in order of likelihood:

	// 1. BSD-style (FreeBSD, OpenBSD, NetBSD, DragonFly)
	// Use unix package constant (most reliable)
	if _, err := unix.IoctlGetTermios(int(fd), unix.TIOCGETA); err == nil {
		return true
	}

	// 2. POSIX-style (AIX, Android)
	if _, err := unix.IoctlGetTermios(int(fd), TCGETS_FALLBACK); err == nil {
		return true
	}

	// 3. System V style (Solaris, Illumos)
	if _, err := unix.IoctlGetTermios(int(fd), TCGETA_FALLBACK); err == nil {
		return true
	}

	// 4. Older Solaris variants
	if _, err := unix.IoctlGetTermios(int(fd), TCGETA_OLD_FALLBACK); err == nil {
		return true
	}

	// 5. QNX real-time OS
	if _, err := unix.IoctlGetTermios(int(fd), QNX_TCGETA); err == nil {
		return true
	}

	// 6. Embedded systems (routers, IoT devices)
	if _, err := unix.IoctlGetTermios(int(fd), TCGETS_EMBEDDED); err == nil {
		return true
	}

	return false
}

// enableANSI is a no-op on Unix systems (ANSI is natively supported)
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

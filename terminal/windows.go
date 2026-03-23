//go:build windows

package terminal

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"unsafe"
)

// Windows-specific syscalls
var (
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleMode = kernel32.NewProc("GetConsoleMode")
	procSetConsoleMode = kernel32.NewProc("SetConsoleMode")
)

// isTerminal checks if fd is a terminal on Windows
func isTerminal(fd uintptr) bool {
	var st uint32
	r, _, e := syscall.SyscallN(procGetConsoleMode.Addr(), fd, uintptr(unsafe.Pointer(&st)))
	return r != 0 && e == 0
}

// enableANSI enables ANSI escape sequence support on Windows 10+
func enableANSI() bool {
	stdout := os.Stdout.Fd()
	var mode uint32

	// Get current console mode
	r, _, _ := syscall.SyscallN(procGetConsoleMode.Addr(), stdout, uintptr(unsafe.Pointer(&mode)))
	if r == 0 {
		return false
	}

	// Enable virtual terminal processing (ANSI support)
	const ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
	mode |= ENABLE_VIRTUAL_TERMINAL_PROCESSING

	// Set the new mode
	r, _, _ = syscall.SyscallN(procSetConsoleMode.Addr(), stdout, uintptr(mode))
	return r != 0
}

// listenForSignals handles Windows signals (no SIGWINCH, only SIGINT/SIGTERM)
func listenForSignals(ctx context.Context, handler *SignalHandler) {
	stopCh := make(chan os.Signal, 1)

	// Windows only supports SIGINT and SIGTERM, no SIGWINCH
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)

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
		case <-stopCh:
			if handler.onStop != nil {
				handler.onStop()
			}
			return
		}
	}
}

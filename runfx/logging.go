package runfx

import (
	"fmt"
	"os"
	"time"
)

var debugMode = false

// EnableDebug turns on verbose terminal state logging
func EnableDebug() {
	debugMode = true
}

// DebugLog prints debug info if debug mode is enabled
func DebugLog(format string, args ...any) {
	if debugMode {
		fmt.Fprintf(os.Stderr, "[RunFX DEBUG] "+format+"\n", args...)
	}
}

// LogMount logs visual mounting
func LogMount(name string) {
	DebugLog("Mounted visual: %s", name)
}

// LogUnmount logs visual unmounting
func LogUnmount(name string) {
	DebugLog("Unmounted visual: %s", name)
}

// LogRender logs rendering events
func LogRender(name string) {
	DebugLog("Rendered visual: %s", name)
}

func LogTick(tickCount, count int, elapsed time.Duration) {
	if debugMode {
		DebugLog("Tick %d: rendered %d visuals in %s", tickCount, count, elapsed.String())
	}
}

func LogFlush(tickCount int, elapsed time.Duration) {
	if debugMode {
		DebugLog("Flush after tick %d took %s", tickCount, elapsed.String())
	}
}

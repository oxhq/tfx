package runfx

import "errors"

var (
	ErrLoopClosed         = errors.New("runfx: event loop is closed")
	ErrMountFailed        = errors.New("runfx: failed to mount visual")
	ErrNotTTY             = errors.New("runfx: not a TTY environment")
	ErrLoopAlreadyRunning = errors.New("runfx: loop is already running")
	ErrLoopNotRunning     = errors.New("runfx: loop is not running")
)

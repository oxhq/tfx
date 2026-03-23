package runfx

import (
	"context"
	"time"
)

type Visual interface {
	Render() []byte
	OnResize(cols, rows int)
}

// Interactive is a Visual that can respond to keyboard input.
type Interactive interface {
	Visual
	OnKey(key Key) bool // Returns true to stop the loop.
}

// Updatable define el hook opcional de tick.
type Updatable interface {
	Visual
	Tick(now time.Time)
}

// Loop defines the runtime loop for mounting and managing visuals.
type Loop interface {
	Mount(v Visual) (unmount func(), err error)
	Run(ctx context.Context) error
	Stop() error
	IsRunning() bool
}

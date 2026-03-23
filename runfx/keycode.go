package runfx

import (
	"fmt"
)

// KeyCode represents a keyboard input event code.
type KeyCode int

const (
	KeyUnknown KeyCode = iota

	// Special keys
	KeyEnter
	KeyEscape
	KeyBackspace
	KeyTab
	KeySpace
	KeyDelete

	// Arrow keys
	KeyArrowUp
	KeyArrowDown
	KeyArrowLeft
	KeyArrowRight

	// Letter keys
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ

	// Number keys
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9

	// Shortcuts
	KeyCtrlC
	KeyCtrlD
	KeyCtrlZ
)

// String returns a readable name for debugging/logging.
func (k KeyCode) String() string {
	switch k {
	case KeyEnter:
		return "Enter"
	case KeyEscape:
		return "Escape"
	case KeyBackspace:
		return "Backspace"
	case KeyTab:
		return "Tab"
	case KeySpace:
		return "Space"
	case KeyDelete:
		return "Delete"
	case KeyArrowUp:
		return "↑"
	case KeyArrowDown:
		return "↓"
	case KeyArrowLeft:
		return "←"
	case KeyArrowRight:
		return "→"
	case KeyCtrlC:
		return "Ctrl+C"
	case KeyCtrlD:
		return "Ctrl+D"
	case KeyCtrlZ:
		return "Ctrl+Z"
	case KeyA, KeyB, KeyC, KeyD, KeyE, KeyF, KeyG, KeyH, KeyI, KeyJ,
		KeyK, KeyL, KeyM, KeyN, KeyO, KeyP, KeyQ, KeyR, KeyS, KeyT,
		KeyU, KeyV, KeyW, KeyX, KeyY, KeyZ:
		alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		index := int(k - KeyA)
		if index >= 0 && index < len(alphabet) {
			return string(alphabet[index])
		}
		return "Unknown"
	case Key0, Key1, Key2, Key3, Key4, Key5, Key6, Key7, Key8, Key9:
		return fmt.Sprintf("%d", k-Key0)
	default:
		return "Unknown"
	}
}

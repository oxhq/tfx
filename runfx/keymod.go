package runfx

import "fmt"

type Modifier uint8

const (
	ModNone Modifier = 0
	ModCtrl Modifier = 1 << iota
	ModAlt
	ModShift
)

func (m Modifier) Has(mod Modifier) bool {
	return m&mod != 0
}

func (m Modifier) String() string {
	if m == ModNone {
		return "None"
	}
	parts := []string{}
	if m.Has(ModCtrl) {
		parts = append(parts, "Ctrl")
	}
	if m.Has(ModAlt) {
		parts = append(parts, "Alt")
	}
	if m.Has(ModShift) {
		parts = append(parts, "Shift")
	}
	return fmt.Sprintf("%v", parts)
}

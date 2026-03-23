package runfx

// Key represents a single key press, with optional modifier info.
type Key struct {
	Code     KeyCode
	Modifier Modifier
	Rune     rune // useful for printable keys
}

// IsArrow returns true if it's an arrow key.
func (k Key) IsArrow() bool {
	return k.Code >= KeyArrowUp && k.Code <= KeyArrowRight
}

// IsWASD returns true if the key is W/A/S/D
func (k Key) IsWASD() bool {
	return k.Code == KeyW || k.Code == KeyA || k.Code == KeyS || k.Code == KeyD
}

func (k Key) IsPrintable() bool {
	return k.Rune != 0 && k.Code != KeyEnter && k.Code != KeyEscape &&
		k.Code != KeyBackspace && k.Code != KeyTab && k.Code != KeyDelete &&
		!k.IsArrow()
}

// IsNumber returns true if it's Key0–Key9
func (k Key) IsNumber() bool {
	return k.Code >= Key0 && k.Code <= Key9
}

// IsCancel returns true if the key is a cancel key (Escape or Ctrl+C).
func (k Key) IsCancel() bool {
	return k.Code == KeyEscape || k.Code == KeyCtrlC || k.Code == KeyCtrlZ
}

// IsNavigation returns true if the key is a navigation key (Arrow keys, WASD, Tab).
func (k Key) IsNavigation() bool {
	return k.IsArrow() || k.IsWASD() || k.Code == KeyTab
}

// IsAccept returns true if the key is an accept key (Enter, Space).
func (k Key) IsAccept() bool {
	return k.Code == KeyEnter || k.Code == KeySpace
}

// IsSelector returns true if the key is used to select an option (navigation, accept or cancel).
func (k Key) IsSelector() bool {
	return k.IsNavigation() || k.IsAccept() || k.IsCancel()
}

// ToNumber returns the number, or -1 if not numeric
func (k Key) ToNumber() int {
	if k.IsNumber() {
		return int(k.Code - Key0)
	}
	return -1
}

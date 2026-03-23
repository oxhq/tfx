package share

// Level represents logging levels
type Level int

const (
	LevelTrace Level = iota
	LevelDebug
	LevelInfo
	LevelSuccess // Success level for positive outcomes
	LevelWarn
	LevelError
	LevelFatal
	LevelPanic
)

func (l Level) String() string {
	switch l {
	case LevelTrace:
		return "TRACE"
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelSuccess:
		return "SUCCESS"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelFatal:
		return "FATAL"
	case LevelPanic:
		return "PANIC"
	default:
		return "UNKNOWN"
	}
}

// ShortString returns a shorter version of the level string
func (l Level) ShortString() string {
	switch l {
	case LevelTrace:
		return "TRC"
	case LevelDebug:
		return "DBG"
	case LevelInfo:
		return "INF"
	case LevelSuccess:
		return "SUC"
	case LevelWarn:
		return "WRN"
	case LevelError:
		return "ERR"
	case LevelFatal:
		return "FAT"
	case LevelPanic:
		return "PAN"
	default:
		return "UNK"
	}
}

// Emoji returns an emoji representation for each level
func (l Level) Emoji() string {
	switch l {
	case LevelTrace:
		return "🔍"
	case LevelDebug:
		return "🐛"
	case LevelInfo:
		return "ℹ️"
	case LevelSuccess:
		return "✅"
	case LevelWarn:
		return "⚠️"
	case LevelError:
		return "❌"
	case LevelFatal:
		return "💀"
	case LevelPanic:
		return "🚨"
	default:
		return "❓"
	}
}

// Icon returns a simplified icon for each level
func (l Level) Icon() string {
	switch l {
	case LevelTrace:
		return "•"
	case LevelDebug:
		return "◦"
	case LevelInfo:
		return "●"
	case LevelSuccess:
		return "✓"
	case LevelWarn:
		return "!"
	case LevelError:
		return "✗"
	case LevelFatal:
		return "†"
	case LevelPanic:
		return "‼"
	default:
		return "?"
	}
}

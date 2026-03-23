package logfx

import (
	"fmt"
	"maps"
	"os"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
)

// Clean color aliases using the new encoding system
var (
	// Default colors (follow active theme)
	White  = color.White
	Black  = color.Black
	Red    = color.Red
	Green  = color.Green
	Blue   = color.Blue
	Yellow = color.Yellow
	Cyan   = color.Cyan
	Purple = color.Purple
	Orange = color.Orange
	Pink   = color.Pink
	Teal   = color.Teal
	Lime   = color.Lime
	Indigo = color.Indigo

	// Material theme colors (explicit)
	MaterialRed    = color.Material.Red
	MaterialGreen  = color.Material.Green
	MaterialBlue   = color.Material.Blue
	MaterialYellow = color.Material.Yellow
	MaterialPurple = color.Material.Purple
	MaterialOrange = color.Material.Orange

	// Dracula theme colors (explicit)
	DraculaRed    = color.Dracula.Red
	DraculaGreen  = color.Dracula.Green
	DraculaBlue   = color.Dracula.Blue
	DraculaYellow = color.Dracula.Yellow
	DraculaPurple = color.Dracula.Purple
	DraculaPink   = color.Dracula.Pink

	// Nord theme colors (explicit)
	NordRed    = color.Nord.Red
	NordGreen  = color.Nord.Green
	NordBlue   = color.Nord.Blue
	NordYellow = color.Nord.Yellow
	NordPurple = color.Nord.Purple
	NordOrange = color.Nord.Orange
)

// Level aliases for cleaner API
const (
	LevelTrace   = share.LevelTrace
	LevelDebug   = share.LevelDebug
	LevelInfo    = share.LevelInfo
	LevelSuccess = share.LevelSuccess
	LevelWarn    = share.LevelWarn
	LevelError   = share.LevelError
	LevelFatal   = share.LevelFatal
	LevelPanic   = share.LevelPanic
)

// FluentLogger provides a fluent/DSL interface for logging
type FluentLogger struct {
	logger *Logger
	level  share.Level
	fields share.Fields
	error  error
	noOp   bool // Add this field
}

// If creates a new fluent logger for conditional logging
func If(err error) *FluentLogger {
	if err == nil {
		return &FluentLogger{noOp: true, fields: make(share.Fields)} // Return a no-op logger
	}
	return &FluentLogger{
		logger: Log(),
		error:  err,
		fields: make(share.Fields),
	}
}

// IfWith creates a new fluent logger with a specific logger instance
func IfWith(logger *Logger, err error) *FluentLogger {
	return &FluentLogger{
		logger: logger,
		error:  err,
		fields: make(share.Fields),
	}
}

// As sets the logging level for the fluent logger
func (f *FluentLogger) As(level share.Level) *FluentLogger {
	f.level = level
	return f
}

// AsFatal sets the level to Fatal
func (f *FluentLogger) AsFatal() *FluentLogger {
	return f.As(share.LevelFatal)
}

// AsError sets the level to Error
func (f *FluentLogger) AsError() *FluentLogger {
	return f.As(share.LevelError)
}

// AsWarn sets the level to Warn
func (f *FluentLogger) AsWarn() *FluentLogger {
	return f.As(share.LevelWarn)
}

// AsInfo sets the level to Info
func (f *FluentLogger) AsInfo() *FluentLogger {
	return f.As(share.LevelInfo)
}

// AsSuccess sets the level to Success
func (f *FluentLogger) AsSuccess() *FluentLogger {
	return f.As(share.LevelSuccess)
}

// AsDebug sets the level to Debug
func (f *FluentLogger) AsDebug() *FluentLogger {
	return f.As(share.LevelDebug)
}

// AsTrace sets the level to Trace
func (f *FluentLogger) AsTrace() *FluentLogger {
	return f.As(share.LevelTrace)
}

// WithField adds a field to the fluent logger
func (f *FluentLogger) WithField(key string, value any) *FluentLogger {
	f.fields[key] = value
	return f
}

// WithFields adds multiple fields to the fluent logger
func (f *FluentLogger) WithFields(fields share.Fields) *FluentLogger {
	maps.Copy(f.fields, fields)
	return f
}

// WithError adds the error to fields (if not already set via If)
func (f *FluentLogger) WithError(err error) *FluentLogger {
	if err != nil {
		f.error = err
		f.fields["error"] = err.Error()
	}
	return f
}

// Msg logs the message with the configured level and fields
func (f *FluentLogger) Msg(msg string, args ...any) bool {
	if f.noOp || f.error == nil {
		return false
	}

	formattedMsg := fmt.Sprintf(msg, args...)
	if f.error != nil {
		f.fields["error"] = f.error.Error()
		formattedMsg = fmt.Sprintf("%s: %v", formattedMsg, f.error)
	}

	if f.level == share.LevelFatal {
		f.logger.log(f.level, formattedMsg, f.fields)
		os.Exit(1)
	}

	f.logger.log(f.level, formattedMsg, f.fields)
	return true
}

// MsgIf logs the message only if the condition is true
func (f *FluentLogger) MsgIf(condition bool, msg string, args ...any) bool {
	if !condition {
		return false
	}
	return f.Msg(msg, args...)
}

// Msgf is an alias for Msg with better naming for formatted strings
func (f *FluentLogger) Msgf(format string, args ...any) bool {
	return f.Msg(format, args...)
}

// Send logs without a message (useful when fields tell the story)
func (f *FluentLogger) Send() bool {
	return f.Msg("")
}

// Modern badge styles with enhanced visual appeal
type BadgeStyle string

const (
	BadgeStyleModern   BadgeStyle = "modern"   // Modern rounded style
	BadgeStyleClassic  BadgeStyle = "classic"  // Traditional square brackets
	BadgeStyleMinimal  BadgeStyle = "minimal"  // Just colored text
	BadgeStyleEmoji    BadgeStyle = "emoji"    // Emoji-based indicators
	BadgeStyleIcon     BadgeStyle = "icon"     // Simple icon indicators
	BadgeStyleGradient BadgeStyle = "gradient" // Gradient effect
	BadgeStyleNeon     BadgeStyle = "neon"     // Neon-like glow effect
)

// ModernBadge creates a modern styled badge log entry
func ModernBadge(tag string, msg string, opts ...BadgeOption) {
	Log().ModernBadge(tag, msg, opts...)
}

// ModernBadge creates a modern styled badge for the logger
func (l *Logger) ModernBadge(tag string, msg string, opts ...BadgeOption) {
	cfg := defaultBadgeConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	fields := share.Fields{
		"badge":       tag,
		"badge_style": cfg.style,
		"badge_color": cfg.color,
		"bg_color":    cfg.bgColor,
		"bold":        cfg.bold,
		"italic":      cfg.italic,
		"underline":   cfg.underline,
	}

	l.log(cfg.level, msg, fields)
}

// BadgeConfig holds configuration for modern badges
type BadgeConfig struct {
	style     BadgeStyle
	color     color.Color
	bgColor   color.Color
	level     share.Level
	bold      bool
	italic    bool
	underline bool
}

// BadgeOption is a functional option for badge configuration
type BadgeOption func(*BadgeConfig)

func defaultBadgeConfig() BadgeConfig {
	return BadgeConfig{
		style:   BadgeStyleModern,
		color:   White,
		bgColor: Blue,
		level:   share.LevelInfo,
		bold:    true,
	}
}

// WithModernBadgeStyle sets the badge style
func WithModernBadgeStyle(style BadgeStyle) BadgeOption {
	return func(cfg *BadgeConfig) {
		cfg.style = style
	}
}

// WithBadgeColor sets the badge text color
func WithBadgeColor(c color.Color) BadgeOption {
	return func(cfg *BadgeConfig) {
		cfg.color = c
	}
}

// WithBadgeBackground sets the badge background color
func WithBadgeBackground(c color.Color) BadgeOption {
	return func(cfg *BadgeConfig) {
		cfg.bgColor = c
	}
}

// WithBadgeLevel sets the badge logging level
func WithBadgeLevel(level share.Level) BadgeOption {
	return func(cfg *BadgeConfig) {
		cfg.level = level
	}
}

// WithBold enables bold text
func WithBold() BadgeOption {
	return func(cfg *BadgeConfig) {
		cfg.bold = true
	}
}

// WithItalic enables italic text
func WithItalic() BadgeOption {
	return func(cfg *BadgeConfig) {
		cfg.italic = true
	}
}

// WithUnderline enables underlined text
func WithUnderline() BadgeOption {
	return func(cfg *BadgeConfig) {
		cfg.underline = true
	}
}

// Predefined modern badge styles
func SuccessBadge(tag, msg string) {
	ModernBadge(tag, msg,
		WithBadgeColor(White),
		WithBadgeBackground(MaterialGreen),
		WithBadgeLevel(share.LevelSuccess),
		WithBold(),
	)
}

func ErrorBadge(tag, msg string) {
	ModernBadge(tag, msg,
		WithBadgeColor(White),
		WithBadgeBackground(MaterialRed),
		WithBadgeLevel(share.LevelError),
		WithBold(),
	)
}

func WarnBadge(tag, msg string) {
	ModernBadge(tag, msg,
		WithBadgeColor(Black),
		WithBadgeBackground(MaterialYellow),
		WithBadgeLevel(share.LevelWarn),
		WithBold(),
	)
}

func InfoBadge(tag, msg string) {
	ModernBadge(tag, msg,
		WithBadgeColor(White),
		WithBadgeBackground(MaterialBlue),
		WithBadgeLevel(share.LevelInfo),
		WithBold(),
	)
}

func DebugBadge(tag, msg string) {
	ModernBadge(tag, msg,
		WithBadgeColor(White),
		WithBadgeBackground(MaterialPurple),
		WithBadgeLevel(share.LevelDebug),
		WithBold(),
	)
}

// Themed badge collections
func DatabaseBadge(msg string, success bool) {
	if success {
		SuccessBadge("DB", msg)
	} else {
		ErrorBadge("DB", msg)
	}
}

func APIBadge(msg string, success bool) {
	if success {
		SuccessBadge("API", msg)
	} else {
		ErrorBadge("API", msg)
	}
}

func CacheBadge(msg string, success bool) {
	if success {
		InfoBadge("CACHE", msg)
	} else {
		WarnBadge("CACHE", msg)
	}
}

func AuthBadge(msg string, success bool) {
	if success {
		SuccessBadge("AUTH", msg)
	} else {
		ErrorBadge("AUTH", msg)
	}
}

func SystemBadge(msg string) {
	InfoBadge("SYS", msg)
}

func ConfigBadge(msg string) {
	InfoBadge("CFG", msg)
}

func SecurityBadge(msg string, level share.Level) {
	switch level {
	case share.LevelError, share.LevelFatal:
		Log().ModernBadge("SEC", msg,
			WithBadgeColor(White),
			WithBadgeBackground(MaterialRed),
			WithBadgeLevel(share.LevelError),
			WithBold(),
		)
	case share.LevelWarn:
		Log().ModernBadge("SEC", msg,
			WithBadgeColor(Black),
			WithBadgeBackground(MaterialYellow),
			WithBadgeLevel(share.LevelWarn),
			WithBold(),
		)
	default:
		Log().ModernBadge("SEC", msg,
			WithBadgeColor(White),
			WithBadgeBackground(MaterialBlue),
			WithBadgeLevel(share.LevelInfo),
			WithBold(),
		)
	}
}

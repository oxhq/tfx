package writer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/terminal"
)

// ConsoleWriter writes formatted logs to console/terminal
type ConsoleWriter struct {
	output     io.Writer
	options    ConsoleOptions
	detector   *terminal.Detector
	badgeWidth int
	mu         sync.Mutex
}

// Options for console writer
type ConsoleOptions struct {
	Level        share.Level
	Format       share.Format
	Timestamp    bool
	TimeFormat   string
	Theme        color.ColorTheme
	BadgeWidth   int
	BadgeStyle   share.BadgeStyle
	ShowCaller   bool
	ForceColor   bool
	DisableColor bool
}

// NewConsoleWriter creates a new console writer
func NewConsoleWriter(output io.Writer, opts ConsoleOptions) *ConsoleWriter {
	if opts.TimeFormat == "" {
		opts.TimeFormat = "15:04:05"
	}
	if opts.BadgeWidth == 0 {
		opts.BadgeWidth = 5
	}

	return &ConsoleWriter{
		output:     output,
		options:    opts,
		detector:   terminal.NewDetector(output),
		badgeWidth: opts.BadgeWidth,
	}
}

func (w *ConsoleWriter) UpdateOptions(output io.Writer, opts ConsoleOptions) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.output = output
	w.options = opts
	w.badgeWidth = opts.BadgeWidth
	w.detector = terminal.NewDetector(output)
}

// Write writes a log entry to console
func (w *ConsoleWriter) Write(entry *share.Entry) error {
	if entry.Level < share.Level(w.options.Level) {
		return nil
	}

	var output string
	switch w.options.Format {
	case share.FormatBadge:
		output = w.formatBadge(entry)
	case share.FormatJSON:
		output = w.formatJSON(entry)
	case share.FormatText:
		output = w.formatText(entry)
	default:
		output = w.formatBadge(entry)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := fmt.Fprintln(w.output, output)
	return err
}

// formatBadge formats entry as a badge log
func (w *ConsoleWriter) formatBadge(entry *share.Entry) string {
	var parts []string

	// Indentation
	if entry.IndentStr != "" {
		parts = append(parts, entry.IndentStr)
	}

	// Timestamp with universal [HH:MM:SS] styling
	if w.options.Timestamp {
		ts := entry.Timestamp.Format(w.options.TimeFormat)
		if w.supportsColor() && !w.options.DisableColor {
			ts = color.Style(ts, color.ModernGray)
			parts = append(parts, fmt.Sprintf("⏰ [%s] ", ts))
		} else {
			parts = append(parts, fmt.Sprintf("[%s] ", ts))
		}
	}

	// Badge/Level - this is already padded for consistent width
	badge := w.formatBadgeTag(entry)
	parts = append(parts, badge)

	// Caller info; dim on debug level
	if w.options.ShowCaller && entry.Caller != nil {
		raw := fmt.Sprintf("%s:%d", w.shortFilename(entry.Caller.File), entry.Caller.Line)
		if w.supportsColor() && !w.options.DisableColor {
			cfg := color.StyleConfig{
				Text: raw,
				// lighter slate for visibility; always undimmed
				ForeGround: color.ModernSlate,
				Dim:        false,
				Mode:       w.GetColorMode(),
			}
			raw = color.NewStyle(cfg)
		}
		parts = append(parts, fmt.Sprintf(" 📍 %s", raw))
	}

	// Message with proper spacing
	message := entry.Message
	if w.supportsColor() && !w.options.DisableColor {
		message = w.colorizeMessage(entry, message)
	}
	// separate badge and message with tab for consistency
	parts = append(parts, "\t"+message)

	// Fields with clean separation
	if len(entry.Fields) > 0 {
		fieldsStr := w.formatFields(entry.Fields)
		if fieldsStr != "" {
			// fieldsStr contains individual key/value styling
			parts = append(parts, "\t"+fieldsStr)
		}
	}

	return strings.Join(parts, "")
}

// formatBadgeTag formats the badge/level part
func (w *ConsoleWriter) formatBadgeTag(entry *share.Entry) string {
	// Check for a pre-styled badge first. If it exists, use it directly.
	if styledBadge, ok := entry.Fields["badge_styled"].(string); ok {
		// Custom styled badge: render exactly as provided
		return styledBadge
	}

	// Determine tag text and color
	var tag string
	var tagColor color.Color
	if badgeTag, ok := entry.Fields["badge"].(string); ok {
		tag = badgeTag
		if bgColor, ok := entry.Fields["bg_color"].(color.Color); ok {
			tagColor = bgColor
		} else if badgeColor, ok := entry.Fields["badge_color"].(color.Color); ok {
			tagColor = badgeColor
		} else {
			tagColor = w.getLevelColor(entry.Level)
		}
	} else {
		tag = w.getLevelTag(entry.Level)
		tagColor = w.getLevelColor(entry.Level)
	}

	// Emoji prefix for level
	emoji := ""
	if w.supportsColor() && !w.options.DisableColor {
		switch entry.Level {
		case share.LevelSuccess:
			emoji = "✅"
		case share.LevelError:
			emoji = "❌"
		case share.LevelWarn:
			emoji = "⚠️ "
		case share.LevelInfo:
			emoji = "ℹ️ "
		case share.LevelDebug:
			emoji = "🐛"
		case share.LevelTrace:
			emoji = "🔍"
		case share.LevelFatal:
			emoji = "🚨"
		case share.LevelPanic:
			emoji = "💣"
		}
	}

	// Multi-part badge: gray background and accent color for second word
	if w.supportsColor() && !w.options.DisableColor && strings.Contains(tag, " ") {
		parts := strings.SplitN(tag, " ", 2)
		main := parts[0]
		accent := parts[1]
		bg := color.ModernGray
		fgMain := color.ColorWhite
		fgAccent := tagColor
		mode := w.GetColorMode()
		styleMain := color.StyleConfig{
			Text:       " " + emoji + main,
			ForeGround: fgMain,
			Background: bg,
			Bold:       true,
			Dim:        entry.Level == share.LevelInfo,
			Mode:       mode,
		}
		styleAccent := color.StyleConfig{
			Text:       " " + accent + " ",
			ForeGround: fgAccent,
			Background: bg,
			Bold:       true,
			Mode:       mode,
		}
		return color.NewStyle(styleMain) + color.NewStyle(styleAccent)
	}

	// Single badge with fixed width padding
	if w.supportsColor() && !w.options.DisableColor {
		// Modern badge style
		var fgColor color.Color
		switch tagColor {
		case color.ModernYellow, color.ModernCyan:
			fgColor = color.ModernSlate
		case color.ModernGray, color.ModernSlate:
			fgColor = color.ColorWhite
		default:
			fgColor = color.ColorWhite
		}
		// pad tag to badgeWidth for alignment
		padded := fmt.Sprintf("%-*s", w.badgeWidth, tag)
		style := color.StyleConfig{
			Text:       " " + emoji + " " + padded + " ",
			ForeGround: fgColor,
			Background: tagColor,
			Bold:       true,
			Dim:        entry.Level == share.LevelInfo,
			Mode:       w.GetColorMode(),
		}
		return color.NewStyle(style)
	}
	// Fallback when color is not supported (basic badge)
	return emoji + " " + tag + " "
}

// colorizeMessage applies color to the message based on level
func (w *ConsoleWriter) colorizeMessage(entry *share.Entry, message string) string {
	if w.options.DisableColor {
		return message
	}
	var fg color.Color
	if msgType, ok := entry.Fields["type"].(string); ok && msgType == "success" {
		fg = color.ModernGreen
	} else {
		switch entry.Level {
		case share.LevelSuccess:
			fg = color.ModernGreen
		case share.LevelError, share.LevelFatal, share.LevelPanic:
			fg = color.ModernRed
		case share.LevelWarn:
			fg = color.ModernOrange
		case share.LevelDebug:
			fg = color.ModernPurple
		case share.LevelTrace:
			fg = color.ModernSlate
		case share.LevelInfo:
			fg = color.ModernBlue
		default:
			fg = color.ModernSlate
		}
	}
	style := color.StyleConfig{
		Text:       message,
		ForeGround: fg,
		Mode:       w.GetColorMode(),
	}
	return color.NewStyle(style)
}

// getLevelTag returns the tag for a level
func (w *ConsoleWriter) getLevelTag(level share.Level) string {
	switch level {
	case share.LevelTrace:
		return "Trace"
	case share.LevelDebug:
		return "Debug"
	case share.LevelInfo:
		return "Info"
	case share.LevelSuccess:
		return "Success"
	case share.LevelWarn:
		return "Warn"
	case share.LevelError:
		return "Error"
	case share.LevelFatal:
		return "Fatal"
	case share.LevelPanic:
		return "Panic"
	default:
		return "Log"
	}
}

// getLevelColor returns the color for a level
func (w *ConsoleWriter) getLevelColor(level share.Level) color.Color {
	switch level {
	case share.LevelTrace:
		return color.ModernSlate
	case share.LevelDebug:
		return w.options.Theme.Debug
	case share.LevelInfo:
		return w.options.Theme.Info
	case share.LevelSuccess:
		return w.options.Theme.Success
	case share.LevelWarn:
		return w.options.Theme.Warning
	case share.LevelError:
		return w.options.Theme.Error
	case share.LevelFatal:
		return color.ModernRed
	case share.LevelPanic:
		return color.ModernRed
	default:
		return w.options.Theme.Info
	}
}

// formatFields formats the fields for display
func (w *ConsoleWriter) formatFields(fields share.Fields) string {
	if len(fields) == 0 {
		return ""
	}
	keys := make([]string, 0, len(fields))
	for key := range fields {
		if key == "badge" || key == "badge_color" || key == "type" || key == "badge_styled" ||
			key == "badge_style" || key == "bg_color" || key == "bold" || key == "italic" || key == "underline" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var parts []string
	for _, key := range keys {
		value := fields[key]
		// key in gray, value in default color
		// key in gray and value in slate for contrast
		keyStr := key
		valRaw := fmt.Sprintf("%v", value)
		if w.supportsColor() && !w.options.DisableColor {
			// muted slate for key, keep value normal
			cfg := color.StyleConfig{
				Text:       key,
				ForeGround: color.ModernSlate,
				Mode:       w.GetColorMode(),
			}
			keyStr = color.NewStyle(cfg)
		}
		parts = append(parts, fmt.Sprintf("%s=%s", keyStr, valRaw))
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf("🔖 %s", strings.Join(parts, " • "))
}

// formatJSON formats entry as JSON.
func (w *ConsoleWriter) formatJSON(entry *share.Entry) string {
	var buf bytes.Buffer
	buf.WriteByte('{')

	first := true
	writeField := func(key string, value string) {
		if !first {
			buf.WriteByte(',')
		}
		first = false

		keyJSON, _ := json.Marshal(key)
		valueJSON, _ := json.Marshal(value)
		buf.Write(keyJSON)
		buf.WriteByte(':')
		buf.Write(valueJSON)
	}

	writeField("level", entry.Level.String())
	writeField("msg", entry.Message)
	writeField("time", entry.Timestamp.Format("2006-01-02T15:04:05.000Z"))

	keys := make([]string, 0, len(entry.Fields))
	for key := range entry.Fields {
		if key == "badge" || key == "badge_color" {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		writeField(key, fmt.Sprint(entry.Fields[key]))
	}

	if entry.Caller != nil {
		writeField("caller", fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line))
	}

	buf.WriteByte('}')
	return buf.String()
}

// formatText formats entry as plain text
func (w *ConsoleWriter) formatText(entry *share.Entry) string {
	var parts []string

	// Timestamp
	if w.options.Timestamp {
		timestamp := entry.Timestamp.Format("2006-01-02 15:04:05")
		parts = append(parts, timestamp)
	}

	// Level
	parts = append(parts, entry.Level.String())

	// Caller
	if w.options.ShowCaller && entry.Caller != nil {
		caller := fmt.Sprintf("%s:%d", w.shortFilename(entry.Caller.File), entry.Caller.Line)
		parts = append(parts, caller)
	}

	// Message
	parts = append(parts, entry.Message)

	// Fields
	if len(entry.Fields) > 0 {
		fieldsStr := w.formatFields(entry.Fields)
		if fieldsStr != "" {
			parts = append(parts, fieldsStr)
		}
	}

	return strings.Join(parts, " ")
}

// Helper methods
func (w *ConsoleWriter) supportsColor() bool {
	if w.options.ForceColor {
		return true
	}
	if w.options.DisableColor {
		return false
	}
	return w.detector.SupportsANSI()
}

func (w *ConsoleWriter) GetColorMode() color.Mode {
	if w.options.ForceColor {
		return color.ModeTrueColor
	}
	if !w.supportsColor() {
		return color.ModeNoColor
	}

	// Priority order: TrueColor > 256Color > ANSI > NoColor
	detectedMode := w.detector.GetMode()

	// Try TrueColor first (24-bit)
	if detectedMode >= 3 { // Assuming 3+ means TrueColor support
		return color.ModeTrueColor
	}

	// Try 256 color (8-bit)
	if detectedMode >= 2 { // Assuming 2 means 256 color support
		return color.Mode256Color
	}

	// Fallback to ANSI (4-bit)
	if detectedMode >= 1 { // Assuming 1 means basic ANSI support
		return color.ModeANSI
	}

	// No color support
	return color.ModeNoColor
}

func (w *ConsoleWriter) shortFilename(filename string) string {
	parts := strings.Split(filename, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return filename
}

// Close closes the writer (no-op for console)
func (w *ConsoleWriter) Close() error {
	return nil
}

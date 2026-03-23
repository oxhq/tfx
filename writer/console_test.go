package writer

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/terminal"
)

func TestNewConsoleWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{
		TimeFormat: "",
		BadgeWidth: 0,
	}
	writer := NewConsoleWriter(buf, opts)

	if writer == nil {
		t.Fatal("NewConsoleWriter returned nil")
	}
	if writer.output != buf {
		t.Errorf("Expected output to be buf, got %v", writer.output)
	}
	// Check default values
	if writer.options.TimeFormat != "15:04:05" {
		t.Errorf("Expected default TimeFormat, got %s", writer.options.TimeFormat)
	}
	if writer.options.BadgeWidth != 5 {
		t.Errorf("Expected default BadgeWidth, got %d", writer.options.BadgeWidth)
	}
}

func TestConsoleWriterWrite(t *testing.T) {
	tests := []struct {
		name           string
		format         share.Format
		entry          *share.Entry
		expectedSubstr string
	}{
		{
			name:   "Badge Format",
			format: share.FormatBadge,
			entry: &share.Entry{
				Level:     share.LevelInfo,
				Message:   "test message",
				Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			expectedSubstr: "test message",
		},
		{
			name:   "JSON Format",
			format: share.FormatJSON,
			entry: &share.Entry{
				Level:     share.LevelDebug,
				Message:   "debug message",
				Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
				Fields:    share.Fields{"key": "value"},
			},
			expectedSubstr: `"level":"DEBUG","msg":"debug message","time":"2023-01-01T10:00:00.000Z","key":"value"`,
		},
		{
			name:   "Text Format",
			format: share.FormatText,
			entry: &share.Entry{
				Level:     share.LevelWarn,
				Message:   "warning message",
				Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			expectedSubstr: "WARN warning message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			opts := ConsoleOptions{
				Format:    tt.format,
				Timestamp: true,
				Theme:     color.DefaultTheme,
			}
			writer := NewConsoleWriter(buf, opts)
			// Mock the detector to ensure color is off for predictable output
			writer.detector.ForceMode(terminal.ModeNoColor)

			err := writer.Write(tt.entry)
			if err != nil {
				t.Errorf("Write() error = %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.expectedSubstr) {
				t.Errorf("Expected output to contain %q, got %q", tt.expectedSubstr, output)
			}
		})
	}

	// Test with level filtering
	t.Run("Level Filtering", func(t *testing.T) {
		buf := &bytes.Buffer{}
		opts := ConsoleOptions{
			Level: share.LevelInfo,
		}
		writer := NewConsoleWriter(buf, opts)
		writer.detector.ForceMode(terminal.ModeNoColor)

		err := writer.Write(&share.Entry{Level: share.LevelDebug, Message: "debug"})
		if err != nil {
			t.Errorf("Write() error = %v", err)
		}
		if buf.String() != "" {
			t.Errorf("Expected empty output for filtered message, got %q", buf.String())
		}
	})
}

func TestFormatBadge(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{
		Timestamp:    true,
		TimeFormat:   "15:04:05",
		BadgeWidth:   8,
		Theme:        color.DefaultTheme,
		ShowCaller:   true,
		DisableColor: true, // Disable color for predictable output
	}
	writer := NewConsoleWriter(buf, opts)
	writer.detector.ForceMode(terminal.ModeNoColor)

	entry := &share.Entry{
		Level:     share.LevelInfo,
		Message:   "test message",
		Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Caller: &share.CallerInfo{
			File:     "/path/to/main.go",
			Line:     100,
			Function: "main.func1",
		},
		Fields: share.Fields{"custom": "field"},
	}

	output := writer.formatBadge(entry)

	// Check timestamp
	if !strings.Contains(output, "[10:00:00]") {
		t.Errorf("Expected timestamp, got %q", output)
	}
	// Check badge
	if !strings.Contains(output, "Info") {
		t.Errorf("Expected badge, got %q", output)
	}
	// Check caller
	if !strings.Contains(output, "main.go:100") {
		t.Errorf("Expected caller, got %q", output)
	}
	// Check message
	if !strings.Contains(output, "test message") {
		t.Errorf("Expected message, got %q", output)
	}
	// Check fields
	if !strings.Contains(output, "custom=field") {
		t.Errorf("Expected fields, got %q", output)
	}

	// Test with styled badge
	t.Run("Styled Badge", func(t *testing.T) {
		entry.Fields["badge_styled"] = "[STYLED]"
		output := writer.formatBadge(entry)
		if !strings.Contains(output, "[STYLED]") {
			t.Errorf("Expected styled badge, got %q", output)
		}
	})
}

func TestFormatJSON(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{}
	writer := NewConsoleWriter(buf, opts)

	entry := &share.Entry{
		Level:     share.LevelInfo,
		Message:   "json message",
		Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Fields:    share.Fields{"data": 123},
		Caller: &share.CallerInfo{
			File: "file.go",
			Line: 42,
		},
	}

	output := writer.formatJSON(entry)

	decoded := map[string]string{}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Expected valid JSON, got %q: %v", output, err)
	}
	if decoded["level"] != "INFO" {
		t.Errorf("Expected level INFO, got %q", decoded["level"])
	}
	if decoded["msg"] != "json message" {
		t.Errorf("Expected message, got %q", decoded["msg"])
	}
	if decoded["time"] != "2023-01-01T10:00:00.000Z" {
		t.Errorf("Expected timestamp, got %q", decoded["time"])
	}
	if decoded["data"] != "123" {
		t.Errorf("Expected field, got %q", decoded["data"])
	}
	if decoded["caller"] != "file.go:42" {
		t.Errorf("Expected caller, got %q", decoded["caller"])
	}
}

func TestFormatJSONEscapesSpecialCharacters(t *testing.T) {
	writer := NewConsoleWriter(&bytes.Buffer{}, ConsoleOptions{})
	entry := &share.Entry{
		Level:     share.LevelInfo,
		Message:   "line1 \"quoted\"\nline2",
		Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Fields:    share.Fields{"data": "value \"x\"\nnext"},
	}

	output := writer.formatJSON(entry)
	decoded := map[string]string{}
	if err := json.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("Expected valid escaped JSON, got %q: %v", output, err)
	}
	if decoded["msg"] != "line1 \"quoted\"\nline2" {
		t.Errorf("Expected escaped message to round-trip, got %q", decoded["msg"])
	}
	if decoded["data"] != "value \"x\"\nnext" {
		t.Errorf("Expected escaped field to round-trip, got %q", decoded["data"])
	}
	if !strings.Contains(output, `\"quoted\"`) || !strings.Contains(output, `\n`) {
		t.Errorf("Expected escaped JSON output, got %q", output)
	}
}

func TestFormatText(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{
		Timestamp:  true,
		ShowCaller: true,
	}
	writer := NewConsoleWriter(buf, opts)

	entry := &share.Entry{
		Level:     share.LevelDebug,
		Message:   "text message",
		Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Caller: &share.CallerInfo{
			File: "text_file.go",
			Line: 10,
		},
		Fields: share.Fields{"extra": "info"},
	}

	output := writer.formatText(entry)

	if !strings.Contains(output, "2023-01-01 10:00:00") {
		t.Errorf("Expected timestamp, got %q", output)
	}
	if !strings.Contains(output, "DEBUG") {
		t.Errorf("Expected level, got %q", output)
	}
	if !strings.Contains(output, "text_file.go:10") {
		t.Errorf("Expected caller, got %q", output)
	}
	if !strings.Contains(output, "text message") {
		t.Errorf("Expected message, got %q", output)
	}
	if !strings.Contains(output, "extra=info") {
		t.Errorf("Expected fields, got %q", output)
	}
}

func TestFormatBadgeTag(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{
		BadgeWidth: 8,
		Theme:      color.DefaultTheme,
	}
	writer := NewConsoleWriter(buf, opts)

	// Test with custom badge tag and color
	t.Run("Custom Badge", func(t *testing.T) {
		entry := &share.Entry{
			Level:  share.LevelInfo,
			Fields: share.Fields{"badge": "CUSTOM", "badge_color": color.Blue},
		}
		output := writer.formatBadgeTag(entry)
		// Expecting ANSI codes, so just check for the tag content
		if !strings.Contains(output, "CUSTOM") {
			t.Errorf("Expected custom badge tag, got %q", output)
		}
	})

	// Test with level-based badge
	t.Run("Level Badge", func(t *testing.T) {
		entry := &share.Entry{
			Level: share.LevelError,
		}
		output := writer.formatBadgeTag(entry)
		// Expecting ANSI codes, so just check for the tag content
		if !strings.Contains(output, "Error") {
			t.Errorf("Expected level badge tag, got %q", output)
		}
	})

	// Test with styled badge (should return it directly)
	t.Run("Styled Badge", func(t *testing.T) {
		entry := &share.Entry{
			Fields: share.Fields{"badge_styled": "[STYLED]"},
		}
		output := writer.formatBadgeTag(entry)
		if output != "[STYLED]" {
			t.Errorf("Expected styled badge, got %q", output)
		}
	})

	// Test with multi-word tag (should apply special formatting)
	t.Run("Multi-word Tag", func(t *testing.T) {
		writer.options.DisableColor = false          // Enable color for this test
		writer.detector.ForceMode(terminal.ModeANSI) // ANSI support
		entry := &share.Entry{
			Level:  share.LevelSuccess,
			Fields: share.Fields{"badge": "MULTI WORD"},
		}
		output := writer.formatBadgeTag(entry)
		// Check for parts of the multi-word badge
		if !strings.Contains(output, "MULTI") || !strings.Contains(output, "WORD") {
			t.Errorf("Expected multi-word badge, got %q", output)
		}
	})

	// Test with emoji
	t.Run("Emoji Badge", func(t *testing.T) {
		writer.options.DisableColor = false          // Enable color for this test
		writer.detector.ForceMode(terminal.ModeANSI) // ANSI support
		entry := &share.Entry{
			Level: share.LevelSuccess,
		}
		output := writer.formatBadgeTag(entry)
		if !strings.Contains(output, "✅") {
			t.Errorf("Expected emoji, got %q", output)
		}
	})
}

func TestColorizeMessage(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{
		Theme: color.DefaultTheme,
	}
	writer := NewConsoleWriter(buf, opts)
	writer.detector.ForceMode(terminal.ModeANSI) // Enable color

	tests := []struct {
		name          string
		level         share.Level
		msgType       string
		expectedColor string // ANSI escape code for the color
	}{
		{"Success Level", share.LevelSuccess, "", color.ModernGreen.Render(writer.GetColorMode())},
		{"Error Level", share.LevelError, "", color.ModernRed.Render(writer.GetColorMode())},
		{"Warn Level", share.LevelWarn, "", color.ModernOrange.Render(writer.GetColorMode())},
		{"Info Level", share.LevelInfo, "", color.ModernBlue.Render(writer.GetColorMode())},
		{"Debug Level", share.LevelDebug, "", color.ModernPurple.Render(writer.GetColorMode())},
		{"Trace Level", share.LevelTrace, "", color.ModernSlate.Render(writer.GetColorMode())},
		{
			"Success Type Field",
			share.LevelInfo,
			"success",
			color.ModernGreen.Render(writer.GetColorMode()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry := &share.Entry{
				Level:  tt.level,
				Fields: make(share.Fields),
			}
			if tt.msgType != "" {
				entry.Fields["type"] = tt.msgType
			}
			message := "test message"
			coloredMessage := writer.colorizeMessage(entry, message)
			if !strings.Contains(coloredMessage, tt.expectedColor) {
				t.Errorf(
					"Expected message to contain color %q, got %q",
					tt.expectedColor,
					coloredMessage,
				)
			}
			if !strings.Contains(coloredMessage, message) {
				t.Errorf("Expected message content, got %q", coloredMessage)
			}
		})
	}

	// Test with color disabled
	t.Run("Color Disabled", func(t *testing.T) {
		writer.options.DisableColor = true
		entry := &share.Entry{Level: share.LevelInfo, Message: "test"}
		coloredMessage := writer.colorizeMessage(entry, "test message")
		if strings.Contains(coloredMessage, "\x1b") {
			t.Errorf("Expected no ANSI codes when color is disabled, got %q", coloredMessage)
		}
	})
}

func TestGetLevelTag(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewConsoleWriter(buf, ConsoleOptions{})

	tests := []struct {
		level    share.Level
		expected string
	}{
		{share.LevelTrace, "Trace"},
		{share.LevelDebug, "Debug"},
		{share.LevelInfo, "Info"},
		{share.LevelSuccess, "Success"},
		{share.LevelWarn, "Warn"},
		{share.LevelError, "Error"},
		{share.LevelFatal, "Fatal"},
		{share.LevelPanic, "Panic"},
		{share.Level(99), "Log"}, // Default case
	}

	for _, tt := range tests {
		actual := writer.getLevelTag(tt.level)
		if actual != tt.expected {
			t.Errorf("For level %v, expected %q, got %q", tt.level, tt.expected, actual)
		}
	}
}

func TestGetLevelColor(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{
		Theme: color.MaterialTheme,
	}
	writer := NewConsoleWriter(buf, opts)

	tests := []struct {
		level    share.Level
		expected color.Color
	}{
		{share.LevelTrace, color.ModernSlate},
		{share.LevelDebug, color.Material.Purple},
		{share.LevelInfo, color.Material.Blue},
		{share.LevelSuccess, color.Material.Green},
		{share.LevelWarn, color.MaterialAmber},
		{share.LevelError, color.Material.Red},
		{share.LevelFatal, color.ModernRed},
		{share.LevelPanic, color.ModernRed},
		{share.Level(99), color.Material.Blue}, // Default case
	}

	for _, tt := range tests {
		actual := writer.getLevelColor(tt.level)
		if actual != tt.expected {
			t.Errorf("For level %v, expected color %v, got %v", tt.level, tt.expected, actual)
		}
	}
}

func TestFormatFields(t *testing.T) {
	buf := &bytes.Buffer{}
	opts := ConsoleOptions{}
	writer := NewConsoleWriter(buf, opts)
	writer.detector.ForceMode(terminal.ModeANSI) // Enable color

	tests := []struct {
		name     string
		fields   share.Fields
		expected string
	}{
		{
			name:   "Basic Fields",
			fields: share.Fields{"user": "john", "id": 123},
			expected: "🔖 " + color.ModernSlate.Render(
				writer.GetColorMode(),
			) + "id" + color.Reset + "=123 • " +
				color.ModernSlate.Render(
					writer.GetColorMode(),
				) + "user" + color.Reset + "=john",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := writer.formatFields(tt.fields)
			if actual != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, actual)
			}
		})
	}

	// Test with color disabled
	t.Run("Color Disabled", func(t *testing.T) {
		writer.options.DisableColor = true
		fields := share.Fields{"user": "john"}
		actual := writer.formatFields(fields)
		if actual != "🔖 user=john" {
			t.Errorf("Expected \"🔖 user=john\", got %q", actual)
		}
	})
}

func TestSupportsColor(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewConsoleWriter(buf, ConsoleOptions{})

	// Test ForceColor
	writer.options.ForceColor = true
	if !writer.supportsColor() {
		t.Error("Expected supportsColor to be true when ForceColor is true")
	}

	// Test DisableColor
	writer.options.ForceColor = false
	writer.options.DisableColor = true
	if writer.supportsColor() {
		t.Error("Expected supportsColor to be false when DisableColor is true")
	}

	// Test Detector (ANSI support)
	writer.options.DisableColor = false
	writer.detector.ForceMode(terminal.ModeANSI) // ANSI
	if !writer.supportsColor() {
		t.Error("Expected supportsColor to be true when detector supports ANSI")
	}

	writer.detector.ForceMode(terminal.ModeNoColor) // No ANSI
	if writer.supportsColor() {
		t.Error("Expected supportsColor to be false when detector does not support ANSI")
	}
}

func TestGetColorMode(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewConsoleWriter(buf, ConsoleOptions{})

	// Test ForceColor
	writer.options.ForceColor = true
	writer.detector.ForceMode(terminal.ModeNoColor) // No ANSI
	if writer.GetColorMode() != color.ModeTrueColor {
		t.Errorf("Expected TrueColor mode when ForceColor is true, got %v", writer.GetColorMode())
	}

	// Test DisableColor
	writer.options.ForceColor = false
	writer.options.DisableColor = true
	if writer.GetColorMode() != color.ModeNoColor {
		t.Errorf("Expected NoColor mode when DisableColor is true, got %v", writer.GetColorMode())
	}

	// Test Detector modes
	writer.options.DisableColor = false
	writer.detector.ForceMode(terminal.ModeTrueColor) // TrueColor
	if writer.GetColorMode() != color.ModeTrueColor {
		t.Errorf("Expected TrueColor mode, got %v", writer.GetColorMode())
	}

	writer.detector.ForceMode(terminal.Mode256) // 256 Color
	if writer.GetColorMode() != color.Mode256Color {
		t.Errorf("Expected 256Color mode, got %v", writer.GetColorMode())
	}

	writer.detector.ForceMode(terminal.ModeANSI) // ANSI
	if writer.GetColorMode() != color.ModeANSI {
		t.Errorf("Expected ANSI mode, got %v", writer.GetColorMode())
	}

	writer.detector.ForceMode(terminal.ModeNoColor) // No Color
	if writer.GetColorMode() != color.ModeNoColor {
		t.Errorf("Expected NoColor mode, got %v", writer.GetColorMode())
	}
}

func TestShortFilename(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewConsoleWriter(buf, ConsoleOptions{})

	tests := []struct {
		path     string
		expected string
	}{
		{"/a/b/c/file.go", "file.go"},
		{"file.go", "file.go"},
		{"/file.go", "file.go"},
		{"", ""},
		{"/", ""},
	}

	for _, tt := range tests {
		actual := writer.shortFilename(tt.path)
		if actual != tt.expected {
			t.Errorf("For path %q, expected %q, got %q", tt.path, tt.expected, actual)
		}
	}
}

func TestConsoleWriterClose(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewConsoleWriter(buf, ConsoleOptions{})

	err := writer.Close()
	if err != nil {
		t.Errorf("Close() returned an error: %v", err)
	}
}

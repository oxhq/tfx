package share

import (
	"strings"
	"testing"
)

func TestANSISequencesAndHelpers(t *testing.T) {
	t.Parallel()

	codes := []string{
		ANSISeq.Black(),
		ANSISeq.Red(),
		ANSISeq.Green(),
		ANSISeq.Yellow(),
		ANSISeq.Blue(),
		ANSISeq.Magenta(),
		ANSISeq.Cyan(),
		ANSISeq.White(),
		ANSISeq.BrightBlack(),
		ANSISeq.BrightRed(),
		ANSISeq.BrightGreen(),
		ANSISeq.BrightYellow(),
		ANSISeq.BrightBlue(),
		ANSISeq.BrightMagenta(),
		ANSISeq.BrightCyan(),
		ANSISeq.BrightWhite(),
		ANSISeq.BgBlack(),
		ANSISeq.BgRed(),
		ANSISeq.BgGreen(),
		ANSISeq.BgYellow(),
		ANSISeq.BgBlue(),
		ANSISeq.BgMagenta(),
		ANSISeq.BgCyan(),
		ANSISeq.BgWhite(),
		ANSISeq.BgBrightBlack(),
		ANSISeq.BgBrightRed(),
		ANSISeq.BgBrightGreen(),
		ANSISeq.BgBrightYellow(),
		ANSISeq.BgBrightBlue(),
		ANSISeq.BgBrightMagenta(),
		ANSISeq.BgBrightCyan(),
		ANSISeq.BgBrightWhite(),
		ANSISeq.Reset(), ANSISeq.Bold(), ANSISeq.Dim(),
	}

	for _, code := range codes {
		if !strings.HasPrefix(code, "\033[") {
			t.Fatalf("expected ANSI escape sequence, got %q", code)
		}
	}

	if got := ANSISeq.Style("hi", ANSISeq.Green()); got != ANSISeq.Green()+"hi"+Reset {
		t.Fatalf("unexpected styled text: %q", got)
	}
	if got := ANSISeq.Wrap("hi", ANSISeq.White(), ANSISeq.BgBlue()); got != ANSISeq.White()+
		ANSISeq.BgBlue()+"hi"+Reset {
		t.Fatalf("unexpected wrapped text: %q", got)
	}
}

func TestLevelRepresentations(t *testing.T) {
	t.Parallel()

	cases := []struct {
		level Level
		full  string
		short string
		emoji string
		icon  string
	}{
		{LevelTrace, "TRACE", "TRC", "🔍", "•"},
		{LevelDebug, "DEBUG", "DBG", "🐛", "◦"},
		{LevelInfo, "INFO", "INF", "ℹ️", "●"},
		{LevelSuccess, "SUCCESS", "SUC", "✅", "✓"},
		{LevelWarn, "WARN", "WRN", "⚠️", "!"},
		{LevelError, "ERROR", "ERR", "❌", "✗"},
		{LevelFatal, "FATAL", "FAT", "💀", "†"},
		{LevelPanic, "PANIC", "PAN", "🚨", "‼"},
		{Level(99), "UNKNOWN", "UNK", "❓", "?"},
	}

	for _, tc := range cases {
		if got := tc.level.String(); got != tc.full {
			t.Fatalf("String() for %v = %q, want %q", tc.level, got, tc.full)
		}
		if got := tc.level.ShortString(); got != tc.short {
			t.Fatalf("ShortString() for %v = %q, want %q", tc.level, got, tc.short)
		}
		if got := tc.level.Emoji(); got != tc.emoji {
			t.Fatalf("Emoji() for %v = %q, want %q", tc.level, got, tc.emoji)
		}
		if got := tc.level.Icon(); got != tc.icon {
			t.Fatalf("Icon() for %v = %q, want %q", tc.level, got, tc.icon)
		}
	}
}

package logfx

import (
	"fmt"
	"os"
	"strings"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/terminal"
)

// BadgeOptions allows advanced visual effects for badges.
type BadgeOptions struct {
	Foreground color.Color
	Background color.Color
	Gradient   []color.Color
	Blink      bool
	Neon       bool
	Theme      string
	Bold       bool
	Italic     bool
	Underline  bool
}

// BadgeWithOptions renders a badge with advanced visual effects.
func BadgeWithOptions(tag, msg string, opts BadgeOptions, args ...any) {
	mode := resolveBadgeColorMode()
	fg, bg, bold, italic, underline, blink := applyCreativeEffects(opts, mode)
	finalBadge, messageStyle := renderBadge(
		tag,
		fg,
		bg,
		bold,
		italic,
		underline,
		blink,
		mode,
		opts.Gradient,
	)
	formattedMsg := fmt.Sprintf(msg, args...)
	styledMessage := color.NewStyle(color.StyleConfig{
		Text:       formattedMsg,
		ForeGround: messageStyle.ForeGround,
		Bold:       messageStyle.Bold,
		Italic:     messageStyle.Italic,
		Blink:      blink,
		Mode:       messageStyle.Mode,
	})

	GetLogger().log(share.LevelInfo, styledMessage, share.Fields{
		"badge_styled": finalBadge,
	})
}

// Legacy Badge for backward compatibility.
func Badge(tag, msg string, color color.Color, args ...any) {
	GetLogger().Badge(tag, msg, color, args...)
}

func resolveBadgeColorMode() color.Mode {
	colorMode := color.ModeTrueColor
	if detector := terminal.NewDetector(os.Stdout); detector != nil {
		switch detector.GetMode() {
		case 0:
			colorMode = color.ModeNoColor
		case 1:
			colorMode = color.ModeANSI
		case 2:
			colorMode = color.Mode256Color
		default:
			colorMode = color.ModeTrueColor
		}
	}
	return colorMode
}

func renderBadge(
	tag string,
	fg, bg color.Color,
	bold, italic, underline, blink bool,
	mode color.Mode,
	gradient []color.Color,
) (finalBadge string, messageStyle color.StyleConfig) {
	badgeText := fmt.Sprintf(" %s ", tag)
	if len(gradient) > 1 {
		runes := []rune(badgeText)
		gradLen := len(gradient)
		parts := make([]string, len(runes))
		for i, r := range runes {
			index := (i * gradLen) / len(runes)
			parts[i] = color.NewStyle(color.StyleConfig{
				Text:       string(r),
				ForeGround: gradient[index],
				Background: gradient[gradLen-1-index],
				Bold:       bold,
				Italic:     italic,
				Underline:  underline,
				Blink:      blink,
				Mode:       mode,
			})
		}
		return strings.Join(parts, ""), color.StyleConfig{
			ForeGround: gradient[gradLen-1],
			Bold:       bold,
			Italic:     italic,
			Mode:       mode,
		}
	}

	return color.NewStyle(color.StyleConfig{
			Text:       badgeText,
			ForeGround: fg,
			Background: bg,
			Bold:       bold,
			Italic:     italic,
			Underline:  underline,
			Blink:      blink,
			Mode:       mode,
		}), color.StyleConfig{
			ForeGround: fg,
			Bold:       bold,
			Italic:     italic,
			Mode:       mode,
		}
}

// applyCreativeEffects applies creative color schemes for special effects.
func applyCreativeEffects(
	opts BadgeOptions,
	mode color.Mode,
) (fg, bg color.Color, bold, italic, underline, blink bool) {
	fg = opts.Foreground
	bg = opts.Background
	bold = opts.Bold
	italic = opts.Italic
	underline = opts.Underline
	blink = opts.Blink

	if opts.Theme != "" {
		if themeColor, ok := color.MaterialPalette()[opts.Theme]; ok {
			bg = themeColor
		}
	}

	switch {
	case opts.Neon:
		fg = color.NewHex("00FFFF")
		bg = color.NewHex("001122")
		bold = true
		underline = true
	case opts.Blink:
		fg = color.NewHex("FF6B6B")
		bg = color.NewHex("2C1810")
		bold = true
		blink = true
	case opts.Bold && opts.Italic && opts.Underline:
		fg = color.NewHex("FFD700")
		bg = color.NewHex("4B0082")
		bold = true
		italic = true
		underline = true
	}

	return fg, bg, bold, italic, underline, blink
}

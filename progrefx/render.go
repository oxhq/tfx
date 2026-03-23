package progrefx

import (
	"fmt"
	"time"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/terminal"
)

// RenderBar builds a progress bar string using theme colors. The bar
// adapts to non-TTY environments by omitting ANSI sequences when
// necessary. When an effect is enabled on the progress theme, RenderBar
// delegates to the theme to draw the animated bar.
func RenderBar(p *Progress, detector *terminal.Detector) string {
	percent := float64(p.current) / float64(p.total)
	style := p.effectiveStyle()

	labelColor := p.theme.RenderColor(p.theme.LabelColor, detector)
	label := labelColor + p.label + color.Reset

	var bar string
	if p.theme.EffectEnabled && p.effect != EffectNone {
		bar = p.theme.RenderProgress(percent, p.width, p.effect, style, detector)
	} else {
		bar = p.theme.renderSolidProgress(int(percent*float64(p.width)), p.width, style, detector)
	}

	percentColor := p.theme.RenderColor(p.theme.PercentColor, detector)
	percentText := percentColor + fmt.Sprintf("%3d%%", int(percent*100)) + color.Reset

	borderColor := p.theme.RenderColor(p.theme.BorderColor, detector)
	leftBorder := borderColor + "[" + color.Reset
	rightBorder := borderColor + "]" + color.Reset

	result := fmt.Sprintf("\r%s %s%s%s %s", label, leftBorder, bar, rightBorder, percentText)

	if p.ShowETA && p.isStarted && p.current > 0 {
		elapsed := time.Since(p.startTime)
		rate := float64(p.current) / elapsed.Seconds()
		if rate > 0 {
			remaining := float64(p.total-p.current) / rate
			etaColor := p.theme.RenderColor(p.theme.PercentColor, detector)
			eta := etaColor + fmt.Sprintf("ETA: %ds", int(remaining)) + color.Reset
			return result + " " + eta
		}
	}

	return result
}

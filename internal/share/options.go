package share

// BadgeStyle represents the style of a log badge.
type BadgeStyle string

const (
	BadgeStyleDefault  BadgeStyle = "default"
	BadgeStyleModern   BadgeStyle = "modern"
	BadgeStyleClassic  BadgeStyle = "classic"
	BadgeStyleMinimal  BadgeStyle = "minimal"
	BadgeStyleEmoji    BadgeStyle = "emoji"
	BadgeStyleIcon     BadgeStyle = "icon"
	BadgeStyleGradient BadgeStyle = "gradient"
	BadgeStyleNeon     BadgeStyle = "neon"
)

// Option is a functional setter for any struct T.
// Example: func WithText(txt string) Option[Config]
type Option[T any] func(*T)

// ApplyOptions applies a set of options to a given instance.
func ApplyOptions[T any](target *T, opts ...Option[T]) {
	for _, opt := range opts {
		opt(target)
	}
}

package share

// Format represents output format
type Format int

const (
	FormatBadge Format = iota
	FormatJSON
	FormatText
	FormatCustom
)

// Formatter defines the interface for custom formatters
type Formatter interface {
	Format(entry *Entry) ([]byte, error)
}

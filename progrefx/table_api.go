package progrefx

import (
	"io"

	"github.com/oxhq/tfx/internal/share"
	"github.com/oxhq/tfx/runfx"
)

// TableConfig holds configuration for a grid-style table view.
type TableConfig struct {
	Title     string
	Columns   []TableColumn
	Rows      [][]string
	Theme     ProgressTheme
	Style     TableStyle
	Writer    io.Writer
	DetectTTY func() runfx.TTYInfo
}

// DefaultTableConfig returns sensible defaults for a table view.
func DefaultTableConfig() TableConfig {
	return TableConfig{
		Theme:     MaterialTheme,
		Style:     TableStyleUnicode,
		DetectTTY: runfx.DetectTTY,
	}
}

// Table creates a table view using multipath configuration.
func Table(opts ...any) *TableView {
	table, err := TryTable(opts...)
	if err != nil {
		panic(err)
	}
	return table
}

// TryTable creates a table view without panicking on invalid args.
func TryTable(opts ...any) (*TableView, error) {
	cfg, err := share.TryOverloadWithOptions(opts, DefaultTableConfig())
	if err != nil {
		return nil, err
	}
	return newTable(cfg), nil
}

// TableBuilder provides a fluent builder API for tables.
type TableBuilder struct {
	config TableConfig
}

// NewTableBuilder returns a builder with default table configuration.
func NewTableBuilder() *TableBuilder {
	return &TableBuilder{config: DefaultTableConfig()}
}

// Title sets the table title.
func (b *TableBuilder) Title(title string) *TableBuilder {
	b.config.Title = title
	return b
}

// Columns sets the table columns.
func (b *TableBuilder) Columns(columns []TableColumn) *TableBuilder {
	b.config.Columns = cloneColumns(columns)
	return b
}

// Rows sets the initial table rows.
func (b *TableBuilder) Rows(rows [][]string) *TableBuilder {
	b.config.Rows = cloneRows(rows)
	return b
}

// Theme sets the table theme.
func (b *TableBuilder) Theme(theme ProgressTheme) *TableBuilder {
	b.config.Theme = theme
	return b
}

// Style sets the table border style.
func (b *TableBuilder) Style(style TableStyle) *TableBuilder {
	b.config.Style = style
	return b
}

// Writer sets the writer used for TTY detection.
func (b *TableBuilder) Writer(writer io.Writer) *TableBuilder {
	b.config.Writer = writer
	return b
}

// DetectTTY allows providing a custom TTY detection function.
func (b *TableBuilder) DetectTTY(fn func() runfx.TTYInfo) *TableBuilder {
	b.config.DetectTTY = fn
	return b
}

// Build constructs the configured table.
func (b *TableBuilder) Build() *TableView {
	return newTable(b.config)
}

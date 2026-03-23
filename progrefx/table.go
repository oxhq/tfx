package progrefx

import (
	"strings"
	"sync"

	"github.com/oxhq/tfx/color"
	"github.com/oxhq/tfx/runfx"
	"github.com/oxhq/tfx/terminal"
)

// TableAlign controls cell alignment inside a table column.
type TableAlign int

const (
	TableAlignLeft TableAlign = iota
	TableAlignCenter
	TableAlignRight
)

// TableStyle controls the border characters used to render a table.
type TableStyle int

const (
	TableStyleUnicode TableStyle = iota
	TableStyleASCII
)

// TableColumn describes a table column.
type TableColumn struct {
	Title string
	Align TableAlign
}

// TableView renders a grid-style table.
type TableView struct {
	title    string
	columns  []TableColumn
	rows     [][]string
	theme    ProgressTheme
	style    TableStyle
	detector *terminal.Detector
	isTTY    bool

	mu sync.Mutex
}

type tableBorder struct {
	topLeft     string
	topMid      string
	topRight    string
	midLeft     string
	midMid      string
	midRight    string
	bottomLeft  string
	bottomMid   string
	bottomRight string
	horizontal  string
	vertical    string
}

func newTable(cfg TableConfig) *TableView {
	detect := cfg.DetectTTY
	if detect == nil {
		detect = runfx.DetectTTY
	}
	theme := cfg.Theme
	if theme == (ProgressTheme{}) {
		theme = MaterialTheme
	}
	tty := detect()

	return &TableView{
		title:    cfg.Title,
		columns:  cloneColumns(cfg.Columns),
		rows:     cloneRows(cfg.Rows),
		theme:    theme,
		style:    cfg.Style,
		detector: terminal.NewDetector(cfg.Writer),
		isTTY:    tty.IsTTY,
	}
}

func cloneColumns(columns []TableColumn) []TableColumn {
	cloned := make([]TableColumn, len(columns))
	copy(cloned, columns)
	return cloned
}

func cloneRows(rows [][]string) [][]string {
	cloned := make([][]string, len(rows))
	for i := range rows {
		cloned[i] = append([]string(nil), rows[i]...)
	}
	return cloned
}

func (t *TableView) border() tableBorder {
	if t.style == TableStyleASCII {
		return tableBorder{
			topLeft:     "+",
			topMid:      "+",
			topRight:    "+",
			midLeft:     "+",
			midMid:      "+",
			midRight:    "+",
			bottomLeft:  "+",
			bottomMid:   "+",
			bottomRight: "+",
			horizontal:  "-",
			vertical:    "|",
		}
	}

	return tableBorder{
		topLeft:     "┌",
		topMid:      "┬",
		topRight:    "┐",
		midLeft:     "├",
		midMid:      "┼",
		midRight:    "┤",
		bottomLeft:  "└",
		bottomMid:   "┴",
		bottomRight: "┘",
		horizontal:  "─",
		vertical:    "│",
	}
}

func cellWidth(s string) int {
	return len([]rune(s))
}

func padCell(value string, width int, align TableAlign) string {
	padding := width - cellWidth(value)
	if padding <= 0 {
		return value
	}

	switch align {
	case TableAlignRight:
		return strings.Repeat(" ", padding) + value
	case TableAlignCenter:
		left := padding / 2
		right := padding - left
		return strings.Repeat(" ", left) + value + strings.Repeat(" ", right)
	default:
		return value + strings.Repeat(" ", padding)
	}
}

func (t *TableView) columnWidths() []int {
	widths := make([]int, len(t.columns))
	for i := range t.columns {
		widths[i] = cellWidth(t.columns[i].Title)
	}
	for _, row := range t.rows {
		for i := range t.columns {
			var cell string
			if i < len(row) {
				cell = row[i]
			}
			if width := cellWidth(cell); width > widths[i] {
				widths[i] = width
			}
		}
	}
	return widths
}

func (t *TableView) styleText(text string, c color.Color) string {
	if !t.isTTY {
		return text
	}
	return t.theme.RenderColor(c, t.detector) + text + color.Reset
}

func (t *TableView) renderBorder(
	left, mid, right string,
	widths []int,
	border tableBorder,
) string {
	parts := make([]string, 0, len(widths))
	for _, width := range widths {
		parts = append(parts, strings.Repeat(border.horizontal, width+2))
	}
	return t.styleText(left+strings.Join(parts, mid)+right, t.theme.BorderColor)
}

func (t *TableView) renderRow(
	values []string,
	widths []int,
	header bool,
	border tableBorder,
) string {
	cells := make([]string, len(t.columns))
	for i, column := range t.columns {
		var value string
		if i < len(values) {
			value = values[i]
		}
		cells[i] = " " + padCell(value, widths[i], column.Align) + " "
	}

	row := border.vertical + strings.Join(cells, border.vertical) + border.vertical
	if !header || !t.isTTY {
		return row
	}
	return t.styleText(row, t.theme.LabelColor)
}

// Render returns the multi-line table representation.
func (t *TableView) Render() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.columns) == 0 {
		if t.title == "" {
			return ""
		}
		return t.styleText(t.title, t.theme.LabelColor)
	}

	border := t.border()
	widths := t.columnWidths()
	lines := make([]string, 0, len(t.rows)+5)
	if t.title != "" {
		lines = append(lines, t.styleText(t.title, t.theme.LabelColor))
	}

	lines = append(
		lines,
		t.renderBorder(border.topLeft, border.topMid, border.topRight, widths, border),
	)

	headers := make([]string, len(t.columns))
	for i, column := range t.columns {
		headers[i] = column.Title
	}
	lines = append(lines, t.renderRow(headers, widths, true, border))
	lines = append(
		lines,
		t.renderBorder(border.midLeft, border.midMid, border.midRight, widths, border),
	)

	for _, row := range t.rows {
		lines = append(lines, t.renderRow(row, widths, false, border))
	}

	lines = append(
		lines,
		t.renderBorder(border.bottomLeft, border.bottomMid, border.bottomRight, widths, border),
	)
	return strings.Join(lines, "\n")
}

// SetTitle updates the table title.
func (t *TableView) SetTitle(title string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.title = title
}

// SetRows replaces the full row set.
func (t *TableView) SetRows(rows [][]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rows = cloneRows(rows)
}

// AppendRow adds a row to the table.
func (t *TableView) AppendRow(row []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.rows = append(t.rows, append([]string(nil), row...))
}

// Rows returns a copy of the current rows.
func (t *TableView) Rows() [][]string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return cloneRows(t.rows)
}

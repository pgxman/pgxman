package tableprinter

import (
	"strings"

	tp "github.com/cli/go-gh/v2/pkg/tableprinter"
	"github.com/cli/go-gh/v2/pkg/term"
)

type TablePrinter struct {
	tp    tp.TablePrinter
	isTTY bool
}

func (t *TablePrinter) SetHeader(columns ...string) {
	if !t.isTTY {
		return
	}
	for _, col := range columns {
		t.tp.AddField(strings.ToUpper(col))
	}
	t.tp.EndRow()
}

func (t *TablePrinter) AppendBluk(rows [][]string) {
	for _, row := range rows {
		for _, col := range row {
			t.tp.AddField(col)
		}
		t.tp.EndRow()
	}
}

func (t *TablePrinter) Render() error {
	return t.tp.Render()
}

var (
	WithTruncate = tp.WithTruncate
	WithColor    = tp.WithColor
)

func New(term term.Term) *TablePrinter {
	maxWidth := 80
	isTTY := term.IsTerminalOutput()
	if isTTY {
		width, _, _ := term.Size()
		if width != 0 {
			maxWidth = width
		}
	}

	tp := tp.New(term.Out(), isTTY, maxWidth)
	return &TablePrinter{
		tp:    tp,
		isTTY: isTTY,
	}
}

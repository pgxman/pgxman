package tableprinter

import (
	"io"
	"os"
	"strings"

	tp "github.com/cli/go-gh/v2/pkg/tableprinter"
	"golang.org/x/term"
)

type TablePrinter struct {
	tp.TablePrinter
	isTTY bool
}

func (t *TablePrinter) HeaderRow(columns ...string) {
	if !t.isTTY {
		return
	}
	for _, col := range columns {
		t.AddField(strings.ToUpper(col))
	}
	t.EndRow()
}

var (
	WithTruncate = tp.WithTruncate
	WithColor    = tp.WithColor
)

func New(w io.Writer) *TablePrinter {
	maxWidth := 80
	isTTY := isStdoutTTY()
	if isTTY {
		width, _, _ := terminalWidth()
		if width != 0 {
			maxWidth = width
		}
	}

	tp := tp.New(w, isTTY, maxWidth)
	return &TablePrinter{
		TablePrinter: tp,
		isTTY:        isTTY,
	}
}

func isStdoutTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

func terminalWidth() (width, height int, err error) {
	return term.GetSize(int(os.Stdout.Fd()))
}

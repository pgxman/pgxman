package iostreams

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/eiannone/keyboard"
	"github.com/mattn/go-isatty"
)

var (
	ErrAbortPrompt = fmt.Errorf("abort prompt")
	ErrNotTerminal = fmt.Errorf("not a terminal")
)

func IsTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

type IOStreams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (i *IOStreams) IsTerminal() bool {
	return IsTerminal(os.Stdout)
}

func (i *IOStreams) Prompt(msg string, continueChars []rune, continueKeys []keyboard.Key) error {
	if !i.IsTerminal() {
		return ErrNotTerminal
	}

	if err := keyboard.Open(); err != nil {
		return err
	}
	defer keyboard.Close()

	fmt.Fprint(i.Stdout, msg+" ")
	for {
		char, key, err := keyboard.GetKey()
		if err != nil {
			return err
		}

		if slices.Contains(continueChars, char) || slices.Contains(continueKeys, key) {
			fmt.Println()
			return nil
		} else {
			fmt.Println()
			return ErrAbortPrompt
		}
	}
}

func NewIOStreams() *IOStreams {
	return &IOStreams{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

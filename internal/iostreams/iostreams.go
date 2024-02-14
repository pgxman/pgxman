package iostreams

import (
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/eiannone/keyboard"
	"golang.org/x/term"
)

var (
	ErrAbortPrompt = fmt.Errorf("abort prompt")
	ErrNotTerminal = fmt.Errorf("not a terminal")
)

type IOStreams struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

func (i *IOStreams) IsTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
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

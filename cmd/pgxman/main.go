package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pgxman/pgxman/internal/cmd/pgxman"
)

type exitCode int

const (
	exitOK    exitCode = 0
	exitError exitCode = 1
)

func main() {
	code := mainRun()
	os.Exit(int(code))
}

func mainRun() exitCode {
	ctx := context.Background()

	cmd, err := pgxman.Execute(ctx)
	if err != nil {
		printError(os.Stderr, err)
		return exitError
	}

	if !cmd.Runnable() {
		return exitError
	}

	return exitOK
}

func printError(out io.Writer, err error) {
	fmt.Fprintln(out, err)
}

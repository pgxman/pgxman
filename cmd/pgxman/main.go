package main

import (
	"fmt"
	"os"

	"github.com/hydradatabase/pgxman/internal/cmd/pgxman"
)

func main() {
	if err := pgxman.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
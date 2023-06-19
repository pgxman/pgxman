package main

import (
	"fmt"
	"os"

	"github.com/pgxman/pgxman/internal/cmd/pgxmanpack"
)

func main() {
	if err := pgxmanpack.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	"github.com/hydradatabase/pgxman/internal/cmd/pgxmanpack"
)

func main() {
	if err := pgxmanpack.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

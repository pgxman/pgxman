package main

import (
	"fmt"
	"os"

	"github.com/hydradatabase/pgxm/internal/cmd/pgxpack"
)

func main() {
	if err := pgxpack.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

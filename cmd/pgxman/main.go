package main

import (
	"fmt"
	"os"

	"github.com/hydradatabase/pgxm/internal/cmd/pgxm"
)

func main() {
	if err := pgxm.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

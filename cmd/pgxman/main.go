package main

import (
	"os"

	"github.com/pgxman/pgxman/internal/cmd/pgxman"
)

func main() {
	if err := pgxman.Execute(); err != nil {
		os.Exit(1)
	}
}

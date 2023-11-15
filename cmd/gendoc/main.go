package main

import (
	"flag"
	"os"
	"path/filepath"

	cmd "github.com/pgxman/pgxman/internal/cmd/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	var (
		markdown string
		man      string
	)

	flag.StringVar(&markdown, "markdown", "", "Path to output markdown docs")
	flag.StringVar(&man, "man", "", "Path to output man pages")
	flag.Parse()

	logger := log.NewTextLogger()

	if markdown == "" {
		logger.Error("-markdown is required")
		os.Exit(1)
	}
	if man == "" {
		logger.Error("-man is required")
		os.Exit(1)
	}

	rootCmd := cmd.Command()
	rootCmd.DisableAutoGenTag = true

	if err := genMarkdown(rootCmd, markdown); err != nil {
		logger.Error("error generating docs", "err", err)
		os.Exit(1)
	}

	if err := genManPages(rootCmd, filepath.Join(man, "man1")); err != nil {
		logger.Error("error generating man pages", "err", err)
		os.Exit(1)
	}
}

func genMarkdown(cmd *cobra.Command, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return doc.GenMarkdownTree(cmd, dir)
}

func genManPages(cmd *cobra.Command, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	header := &doc.GenManHeader{
		Title:   "pgxman",
		Section: "1",
		Source:  "pgxman",
		Manual:  "PostgreSQL Extension Manager",
	}

	return doc.GenManTree(cmd, header, dir)
}

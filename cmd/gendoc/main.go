package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/pgxman/pgxman"
	cmd "github.com/pgxman/pgxman/internal/cmd/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	var (
		docs string
		man  string
	)

	flag.StringVar(&docs, "docs", "", "Path to output docs")
	flag.StringVar(&man, "man", "", "Path to output man pages")
	flag.Parse()

	logger := log.NewTextLogger()

	if docs == "" {
		logger.Error("-docs is required")
		os.Exit(1)
	}
	if man == "" {
		logger.Error("-man is required")
		os.Exit(1)
	}

	rootCmd := cmd.Command()

	if err := genDocs(rootCmd, docs); err != nil {
		logger.Error("error generating docs", "err", err)
		os.Exit(1)
	}

	if err := genManPages(rootCmd, filepath.Join(man, "man", "man1")); err != nil {
		logger.Error("error generating man pages", "err", err)
		os.Exit(1)
	}

	if err := genCompletionScripts(rootCmd, filepath.Join(man, "completion")); err != nil {
		logger.Error("error generating completion scripts", "err", err)
		os.Exit(1)
	}

}

func genDocs(cmd *cobra.Command, dir string) error {
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
		Source:  "pgxman " + pgxman.Version,
		Manual:  "PostgreSQL Extension Manager",
	}

	return doc.GenManTree(cmd, header, dir)
}

func genCompletionScripts(cmd *cobra.Command, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := cmd.GenBashCompletionFile(filepath.Join(dir, "pgxman.bash_completion.sh")); err != nil {
		return err
	}

	if err := cmd.GenZshCompletionFile(filepath.Join(dir, "pgxman.zsh_completion")); err != nil {
		return err
	}

	return nil
}

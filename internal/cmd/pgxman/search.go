package pgxman

import (
	"fmt"
	"regexp"

	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/tableprinter"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search [<query>]",
		Aliases: []string{"s"},
		Short:   "Search for extensions",
		Long: `Search for installable PostgreSQL extensions. The query is a regular expression that is matched
against the extension name and description.`,
		Example: `  # Search for pgvector
  pgxman search pgvector

  # Search by regular expression
  pgxman search ^pg_
		`,
		Args: cobra.MinimumNArgs(1),
		RunE: runSearch,
	}

	return cmd
}

func runSearch(c *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no search terms provided")
	}

	re, err := regexp.Compile(args[0])
	if err != nil {
		return err
	}

	exts, err := buildkit.Extensions()
	if err != nil {
		return err
	}

	if len(exts) == 0 {
		fmt.Println("No extensions found.")
		return nil
	}

	tp := tableprinter.New(term.FromEnv())
	tp.SetHeader("Name", "Version", "Description")

	var rows [][]string
	for _, ext := range exts {
		if re.MatchString(ext.Name) || re.MatchString(ext.Description) {
			rows = append(rows, []string{ext.Name, ext.Version, ext.Description})
		}
	}
	tp.AppendBluk(rows)

	return tp.Render()
}

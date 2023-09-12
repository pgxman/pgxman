package pgxman

import (
	"fmt"
	"os"
	"regexp"

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

  # Search by reguard expression
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

	exts, err := buildkit.InstallableExtensions(c.Context())
	if err != nil {
		return err
	}

	if len(exts) == 0 {
		fmt.Println("No extensions found.")
		return nil
	}

	tp := tableprinter.New(os.Stdout)
	tp.HeaderRow("Name", "Version", "Description")

	for _, ext := range exts {
		if re.MatchString(ext.Name) || re.MatchString(ext.Description) {
			tp.AddField(ext.Name)
			tp.AddField(ext.Version)
			tp.AddField(ext.Description)
			tp.EndRow()
		}
	}

	return tp.Render()
}

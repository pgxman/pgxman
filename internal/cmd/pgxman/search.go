package pgxman

import (
	"fmt"

	"github.com/cli/go-gh/v2/pkg/term"
	"github.com/pgxman/pgxman/internal/tui/tableprinter"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search [<query>]",
		Aliases: []string{"s"},
		Short:   "Search for PostgreSQL extensions",
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

func runSearch(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no search terms provided")
	}

	client, err := newReigstryClient()
	if err != nil {
		return err
	}

	exts, err := client.FindExtension(cmd.Context(), args)
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
		rows = append(rows, []string{ext.Name, ext.Version, ext.Description})
	}
	tp.AppendBluk(rows)

	return tp.Render()
}

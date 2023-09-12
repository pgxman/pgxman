package pgxman

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "search [search terms ...]",
		Aliases: []string{"s"},
		Short:   "Search for extensions",
		Long:    `Search for installable PostgreSQL extensions.`,
		Example: `  # Search for pgvector
  pgxman search pgvector
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

	return nil
}

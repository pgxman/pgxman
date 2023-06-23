package pgxman

import (
	"github.com/pgxman/pgxman"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Fetch the newest version of all extensions",
		RunE:  runUpdate,
	}
	return cmd
}

func runUpdate(c *cobra.Command, args []string) error {
	return pgxman.NewUpdater().Update(c.Context())
}

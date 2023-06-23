package pgxman

import (
	"github.com/pgxman/pgxman/internal/plugin"
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
	u, err := plugin.GetUpdater()
	if err != nil {
		return err
	}

	return u.Update(c.Context())
}

package pgxman

import (
	"fmt"

	"github.com/pgxman/pgxman/internal/errorsx"
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
		return errorsx.Pretty(err)
	}

	if err := u.Update(c.Context()); err != nil {
		return err
	}

	fmt.Println("Update complete.")
	return nil
}

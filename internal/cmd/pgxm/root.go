package pgxm

import (
	"github.com/hydradatabase/pgxm/internal/cmd"
	"github.com/spf13/cobra"
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxm",
		Short:   "PostgreSQL Extension Manager",
		Version: cmd.Version,
	}

	root.AddCommand(newPackageCmd())

	return root.Execute()
}

package pgxm

import (
	"github.com/hydradatabase/pgxm"
	"github.com/spf13/cobra"
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxm",
		Short:   "PostgreSQL Extension Manager",
		Version: pgxm.Version,
	}

	root.AddCommand(newBuildCmd())

	return root.Execute()
}

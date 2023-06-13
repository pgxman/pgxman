package pgxman

import (
	"os"

	pgxm "github.com/hydradatabase/pgxman"
	"github.com/hydradatabase/pgxman/internal/log"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

var (
	flagDebug bool
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxman",
		Short:   "PostgreSQL Extension Manager",
		Version: pgxm.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagDebug {
				log.SetLevel(slog.LevelDebug)
			}
		},
	}

	root.AddCommand(newInitCmd())
	root.AddCommand(newBuildCmd())

	root.PersistentFlags().BoolVar(&flagDebug, "debug", os.Getenv("DEBUG") != "", "enable debug logging")

	return root.Execute()
}

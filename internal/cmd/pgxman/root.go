package pgxman

import (
	"os"

	"log/slog"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/spf13/cobra"
)

var (
	flagDebug bool
)

func Execute() error {
	root := &cobra.Command{
		Use:          "pgxman",
		Short:        "PostgreSQL Extension Manager",
		Version:      pgxman.Version,
		SilenceUsage: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagDebug {
				log.SetLevel(slog.LevelDebug)
			}
		},
	}

	root.AddCommand(newInitCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newBuildCmd())
	root.AddCommand(newInstallCmd())
	root.AddCommand(newUpgradeCmd())
	root.AddCommand(newBundleCmd())
	root.AddCommand(newPublishCmd())
	root.AddCommand(newContainerCmd())
	root.AddCommand(newDoctorCmd())

	root.PersistentFlags().BoolVar(&flagDebug, "debug", os.Getenv("DEBUG") != "", "enable debug logging")

	return root.Execute()
}

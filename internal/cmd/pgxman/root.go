package pgxman

import (
	"os"

	"log/slog"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/spf13/cobra"
)

var (
	flagDebug       bool
	flagRegistryURL string
)

func Command() *cobra.Command {
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
	root.AddCommand(newPackCmd())
	root.AddCommand(newPublishCmd())
	root.AddCommand(newContainerCmd())
	root.AddCommand(newDoctorCmd())

	root.PersistentFlags().BoolVar(&flagDebug, "debug", os.Getenv("DEBUG") != "", "enable debug logging")
	root.PersistentFlags().StringVar(&flagRegistryURL, "registry", "https://registry.pgxman.com/v1", "registry URL")

	return root
}

func Execute() error {
	return Command().Execute()
}

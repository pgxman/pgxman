package pgxman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"

	"log/slog"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/pg"
	"github.com/pgxman/pgxman/internal/registry"
	"github.com/pgxman/pgxman/internal/upgrade"
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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flagDebug {
				log.SetLevel(slog.LevelDebug)
			}

			return checkUpgrade(cmd.Context())
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
	root.AddCommand(newAuthCmd())

	root.PersistentFlags().BoolVar(&flagDebug, "debug", os.Getenv("DEBUG") != "", "enable debug logging")
	root.PersistentFlags().StringVar(&flagRegistryURL, "registry", "https://registry.pgxman.com/v1", "registry URL")

	return root
}

func Execute() error {
	return Command().Execute()
}

func checkUpgrade(ctx context.Context) error {
	c := upgrade.NewChecker(log.NewTextLogger())
	result, err := c.Check(ctx)
	if err != nil {
		return err
	}

	if result.HasUpgrade {
		var upgradeCmd string
		switch runtime.GOOS {
		case "darwin":
			upgradeCmd = "brew upgrade pgxman"
		case "linux":
			upgradeCmd = "apt upgrade pgxman"
		default:
			// skip upgrade message for unsupported platform
			return nil
		}

		msg := fmt.Sprintf("pgxman %s available (%s installed), run `%s` to upgrade", result.LatestVersion, result.CurrentVersion, upgradeCmd)
		fmt.Println(infoColor.Copy().SetString(msg))
	}

	return nil
}

var (
	errCanNotDetectPG = errors.New("could not detect a supported installation of PostgreSQL. For more info, run `pgxman doctor`")
)

func supportedPGVersions() []string {
	var pgVers []string
	for _, v := range pgxman.SupportedPGVersions {
		pgVers = append(pgVers, string(v))
	}

	return pgVers
}

func checkPGVerExists(ctx context.Context, pgVer pgxman.PGVersion) error {
	if pgVer == pgxman.PGVersionUnknown || !pg.VersionExists(ctx, pgVer) {
		return errCanNotDetectPG
	}

	return nil
}

func newReigstryClient(token string) (registry.Client, error) {
	return registry.NewClient(flagRegistryURL, token)
}

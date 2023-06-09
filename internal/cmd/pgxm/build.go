package pgxm

import (
	"os"
	"path/filepath"

	"github.com/hydradatabase/pgxm"
	"github.com/hydradatabase/pgxm/internal/cmd"
	"github.com/spf13/cobra"
)

var (
	flagSet map[string]string
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build extension according to the configuration file",
		RunE:  runBuild,
	}

	cmd.PersistentFlags().StringToStringVarP(&flagSet, "set", "s", map[string]string{}, "override values in the extension.yaml file in the format of --set KEY=VALUE, e.g. --set version=1.0.0 --set arch=[amd64,arm64] --set pgVersions=[10,11,12]")

	return cmd
}

func runBuild(c *cobra.Command, args []string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	overrides := cmd.ParseMapFlag(flagSet)

	ext, err := pgxm.ReadExtensionFile(filepath.Join(pwd, "extension.yaml"), overrides)
	if err != nil {
		return err
	}

	builder := pgxm.NewBuilder(pwd)
	return builder.Build(c.Context(), ext)
}

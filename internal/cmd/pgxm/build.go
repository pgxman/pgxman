package pgxm

import (
	"os"
	"path/filepath"

	"github.com/hydradatabase/pgxm"
	"github.com/spf13/cobra"
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build extension according to the configuration file",
		RunE:  runBuild,
	}

	return cmd
}

func runBuild(cmd *cobra.Command, args []string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	ext, err := pgxm.ReadExtensionFile(filepath.Join(pwd, "extension.yaml"))
	if err != nil {
		return err
	}

	builder := pgxm.NewBuilder()
	return builder.Build(cmd.Context(), ext)
}

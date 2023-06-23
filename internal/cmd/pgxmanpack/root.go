package pgxmanpack

import (
	"os"
	"path/filepath"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"
)

var (
	flagDebug bool
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxman-pack",
		Short:   "PostgreSQL Extension Packager",
		Version: pgxman.Version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagDebug {
				log.SetLevel(slog.LevelDebug)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd, err := os.Getwd()
			if err != nil {
				return err
			}

			ext, err := pgxman.ReadExtension(filepath.Join(pwd, "extension.yaml"), nil)
			if err != nil {
				return err
			}

			pkg, err := plugin.GetPackager()
			if err != nil {
				return err
			}

			if err := pkg.Package(
				cmd.Context(),
				ext,
				pgxman.PackagerOptions{
					WorkDir: pwd,
					Debug:   flagDebug,
				},
			); err != nil {
				return err
			}

			return nil
		},
	}

	root.PersistentFlags().BoolVar(&flagDebug, "debug", os.Getenv("DEBUG") != "", "enable debug logging")

	return root.Execute()
}

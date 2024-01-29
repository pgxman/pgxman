package pgxmanpack

import (
	"fmt"
	"os"
	"path/filepath"

	"log/slog"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	flagDebug    bool
	flagParallel int

	extension    pgxman.Extension
	packager     pgxman.Packager
	packagerOpts pgxman.PackagerOptions
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxman-pack",
		Short:   "PostgreSQL Extension Packager",
		Version: pgxman.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if flagDebug {
				log.SetLevel(slog.LevelDebug)
			}

			if flagParallel < 1 {
				return fmt.Errorf("invalid parallel value: %d", flagParallel)
			}

			var err error
			packager, err = plugin.GetPackager()
			if err != nil {
				return errorsx.Pretty(err)
			}

			workDir, err := os.Getwd()
			if err != nil {
				return err
			}

			extension, err = pgxman.ReadExtension(filepath.Join(workDir, "extension.yaml"), nil)
			if err != nil {
				return err
			}

			packagerOpts = pgxman.PackagerOptions{
				WorkDir:  workDir,
				Parallel: flagParallel,
				Debug:    flagDebug,
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runInit(cmd, args); err != nil {
				return err
			}

			if err := runPre(cmd, args); err != nil {
				return err
			}

			if err := runMain(cmd, args); err != nil {
				return err
			}

			if err := runPost(cmd, args); err != nil {
				return err
			}

			return nil
		},
	}

	root.PersistentFlags().BoolVar(&flagDebug, "debug", os.Getenv("DEBUG") != "", "enable debug logging")
	root.PersistentFlags().IntVar(&flagParallel, "parallel", 2, "number of parallel builds to run")

	root.AddCommand(newInitCmd())
	root.AddCommand(newPreCmd())
	root.AddCommand(newMainCmd())
	root.AddCommand(newPostCmd())

	return root.Execute()
}

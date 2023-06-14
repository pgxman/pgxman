package pgxmanpack

import (
	"os"
	"path/filepath"

	"github.com/hydradatabase/pgxman"
	"github.com/hydradatabase/pgxman/internal/log"
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

			pack := pgxman.NewPackager(pwd, flagDebug)
			if err := pack.Package(cmd.Context(), ext); err != nil {
				return err
			}

			return nil
		},
	}

	root.PersistentFlags().BoolVar(&flagDebug, "debug", os.Getenv("DEBUG") != "", "enable debug logging")

	return root.Execute()
}

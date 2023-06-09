package pgxpack

import (
	"os"
	"path/filepath"

	"github.com/hydradatabase/pgxm"
	"github.com/spf13/cobra"
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxpack",
		Short:   "PostgreSQL Extension Packager",
		Version: pgxm.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd, err := os.Getwd()
			if err != nil {
				return err
			}

			ext, err := pgxm.ReadExtensionFile(filepath.Join(pwd, "extension.yaml"))
			if err != nil {
				return err
			}

			pack := pgxm.NewPackager(pwd)
			return pack.Package(cmd.Context(), ext)
		},
	}

	return root.Execute()
}

package pgxmanpack

import (
	"os"
	"path/filepath"

	"github.com/hydradatabase/pgxman"
	"github.com/spf13/cobra"
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxpack",
		Short:   "PostgreSQL Extension Packager",
		Version: pgxman.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			pwd, err := os.Getwd()
			if err != nil {
				return err
			}

			ext, err := pgxman.ReadExtensionFile(filepath.Join(pwd, "extension.yaml"), nil)
			if err != nil {
				return err
			}

			pack := pgxman.NewPackager(pwd)
			return pack.Package(cmd.Context(), ext)
		},
	}

	return root.Execute()
}

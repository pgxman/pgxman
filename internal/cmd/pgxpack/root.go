package pgxpack

import (
	"fmt"
	"os"

	"github.com/hydradatabase/pgxm"
	"github.com/hydradatabase/pgxm/internal/cmd"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

func Execute() error {
	root := &cobra.Command{
		Use:     "pgxpack",
		Short:   "PostgreSQL Extension Packager",
		Version: cmd.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			extFile := "extension.yaml"
			if _, err := os.Stat(extFile); err != nil {
				return fmt.Errorf("extension.yaml not found in current directory")
			}

			b, err := os.ReadFile(extFile)
			if err != nil {
				return err
			}

			var ext pgxm.Extension
			if err := yaml.Unmarshal(b, &ext); err != nil {
				return err
			}

			pwd, err := os.Getwd()
			if err != nil {
				return err
			}

			pack := pgxm.NewPackager(pwd)
			return pack.Package(cmd.Context(), ext)
		},
	}

	return root.Execute()
}

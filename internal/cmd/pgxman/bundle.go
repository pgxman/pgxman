package pgxman

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	flagBundleYes        bool
	flagBundleSudo       bool
	flagBundlePGXManfile string
)

func newBundleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bundle",
		Short: "Manage PostgreSQL extensions from a bundle file",
		Long: `Install or upgrade PostgreSQL extensions based on a specified bundle file (e.g., pgxman.yaml).
This ensures consistency across extensions by synchronizing them with the definitions provided in the bundle file.`,
		Example: `  # Install or upgrade extensions from the pgxman.yaml file in the current directory
  pgxman bundle

  # Suppress prompts for automatic installation or upgrade
  pgxman bundle -y

  # Specify a different location for the pgxman.yaml file
  pgxman bundle -f /PATH_TO/pgxman.yaml

  # Read the pgxman.yaml file from STDIN
  cat <<EOF | pgxman bundle -f -
    apiVersion: v1
    extensions:
      - name: "pgvector"
        version: "0.5.0"
      - path: "/local/path/to/extension"
    postgres:
      version: "14"
  EOF
  `,
		RunE: runBundle,
	}

	cmd.PersistentFlags().StringVarP(&flagBundlePGXManfile, "file", "f", "", "The pgxman.yaml file to use. Defaults to pgxman.yaml in the current directory.")
	cmd.PersistentFlags().BoolVar(&flagBundleSudo, "sudo", os.Getenv("PGXMAN_SUDO") != "", "Run the underlaying package manager command with sudo.")
	cmd.PersistentFlags().BoolVarP(&flagBundleYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)

	return cmd
}

func runBundle(c *cobra.Command, args []string) error {
	i, err := plugin.GetInstaller()
	if err != nil {
		return errorsx.Pretty(err)
	}

	if flagBundlePGXManfile == "" {
		pwd, err := os.Getwd()
		if err != nil {
			return err
		}

		flagBundlePGXManfile = filepath.Join(pwd, "pgxman.yaml")
	}

	f, err := pgxman.ReadPGXManfile(flagBundlePGXManfile)
	if err != nil {
		return err
	}

	pgVer := f.Postgres.Version
	if err := validatePGVer(c.Context(), pgVer); err != nil {
		return err
	}

	if err := LockPGXManfile(f, log.NewTextLogger()); err != nil {
		return err
	}

	s := newSpinner()
	defer s.Stop()

	s.Suffix = fmt.Sprintf(" Bundling extensions for PostgreSQL %s...\n", pgVer)
	s.FinalMSG = extOutput(f)

	opts := []pgxman.InstallerOptionsFunc{
		pgxman.WithSudo(flagBundleSudo),
	}
	if flagBundleYes {
		s.Start()
	} else {
		opts = append(opts, pgxman.WithBeforeRunHook(func(debPkgs []string, sources []string) error {
			if err := promptInstallOrUpgrade(debPkgs, sources, false); err != nil {
				return err
			}

			s.Start()
			return nil
		}))
	}

	if err := i.Upgrade(
		c.Context(),
		*f,
		opts...,
	); err != nil {
		return fmt.Errorf("failed to bundle extensions, run with `--debug` to see the full error")
	}

	return nil
}

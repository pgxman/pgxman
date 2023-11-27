package pgxman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	flagBundleYes  bool
	flagBundleFile string
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

	pwd, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}

	cmd.PersistentFlags().StringVarP(&flagBundleFile, "file", "f", filepath.Join(pwd, "pgxman.yaml"), "The bundle file to use.")
	cmd.PersistentFlags().BoolVarP(&flagBundleYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)

	return cmd
}

func runBundle(cmd *cobra.Command, args []string) error {
	i, err := plugin.GetInstaller()
	if err != nil {
		return errorsx.Pretty(err)
	}

	b, err := pgxman.ReadBundleFile(flagBundleFile)
	if err != nil {
		return err
	}

	if err := b.Validate(); err != nil {
		return err
	}

	pgVer := b.Postgres.Version
	if err := checkPGVerExists(cmd.Context(), pgVer); err != nil {
		return err
	}

	exts, err := LockExtensions(installExts(*b), log.NewTextLogger())
	if err != nil {
		return err
	}

	if !flagBundleYes {
		if err := i.PreInstallCheck(cmd.Context(), exts, pgxman.NewStdIO()); err != nil {
			return err
		}
	}

	fmt.Printf("Bundling extensions for PostgreSQL %s...\n", pgVer)
	for _, ext := range exts {
		if err := installOrUpgrade(cmd.Context(), i, ext, true); err != nil {
			// Error message is already shown in spinner
			os.Exit(1)
		}
	}

	return nil
}

func installOrUpgrade(ctx context.Context, i pgxman.Installer, ext pgxman.InstallExtension, upgrade bool) error {
	s := newSpinner()
	s.Suffix = fmt.Sprintf(" Installing %s...\n", ext)
	defer s.Stop()

	f := i.Install
	if upgrade {
		f = i.Upgrade
	}

	handleErr := func(err error) error {
		if errors.Is(err, pgxman.ErrRootAccessRequired) {
			return fmt.Errorf("must run command as root: sudo %s", strings.Join(os.Args, " "))
		}

		if errors.Is(err, pgxman.ErrConflictExtension) {
			return fmt.Errorf("has already been installed (outside of pgxman)")
		}

		return fmt.Errorf("failed to install, run with `--debug` to see the full error: %w", err)
	}

	s.Start()
	if err := f(ctx, ext); err != nil {
		err = handleErr(err)
		s.FinalMSG = fmt.Sprintf("[%s] %s: %s\n", errorMark, ext, err)
		return err
	}

	s.FinalMSG = fmt.Sprintf("[%s] %s: https://pgx.sh/%s\n", successMark, ext, ext.Name)

	return nil
}

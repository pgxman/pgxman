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
	"github.com/pgxman/pgxman/internal/tui/spinner"
	"github.com/spf13/cobra"
)

var (
	flagPackInstallYes  bool
	flagPackInstallFile string
)

func newPackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Manage PostgreSQL extensions from a pack file",
	}

	cmd.AddCommand(newPackInstallCmd())

	return cmd
}

func newPackInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install PostgreSQL extensions from a pack file",
		Long: `Install PostgreSQL extensions based on a specified pack file (e.g., pgxman.yaml).
This ensures consistency across extensions by synchronizing them with the definitions provided in the pack file.`,
		Example: `  # Install extensions from the pgxman.yaml file in the current directory
  pgxman pack install

  # Suppress prompts for automatic installation or upgrade
  pgxman pack install -y

  # Specify a different location for the pgxman.yaml file
  pgxman pack install -f /PATH_TO/pgxman.yaml

  # Read the pgxman.yaml file from STDIN
  cat <<EOF | pgxman pack install -f -
    apiVersion: v1
    extensions:
      - name: "pgvector"
        version: "0.5.0"
      - path: "/local/path/to/extension"
    postgres:
      version: "14"
  EOF
  `,
		RunE: runPackInstall,
	}

	pwd, err := os.Getwd()
	if err != nil {
		panic(err.Error())
	}

	cmd.PersistentFlags().StringVarP(&flagPackInstallFile, "file", "f", filepath.Join(pwd, "pgxman.yaml"), "The pack file to use.")
	cmd.PersistentFlags().BoolVarP(&flagPackInstallYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)

	return cmd
}

func runPackInstall(cmd *cobra.Command, args []string) error {
	i, err := plugin.GetInstaller()
	if err != nil {
		return errorsx.Pretty(err)
	}

	p, err := pgxman.ReadPackFile(flagPackInstallFile)
	if err != nil {
		return err
	}
	if err := p.Validate(); err != nil {
		return err
	}

	pgVer := p.Postgres.Version
	if err := checkPGVerExists(cmd.Context(), pgVer); err != nil {
		return err
	}

	client, err := newReigstryClient()
	if err != nil {
		return err
	}

	locker := NewExtensionLocker(client, DefaultPlatformDetector, log.NewTextLogger())
	exts, err := locker.Lock(cmd.Context(), p.InstallExtensions())
	if err != nil {
		return err
	}

	if !flagPackInstallYes {
		if err := i.PreInstallCheck(cmd.Context(), exts, pgxman.NewStdIO()); err != nil {
			return err
		}
	}

	logger := log.NewTextLogger()
	fmt.Printf("Bundling extensions for PostgreSQL %s...\n", pgVer)
	for _, ext := range exts {
		if err := installOrUpgrade(cmd.Context(), i, ext, true); err != nil {
			logger.Debug("failed to install extension", "error", err)
			os.Exit(1)
		}
	}

	return nil
}

func installOrUpgrade(ctx context.Context, i pgxman.Installer, ext pgxman.InstallExtension, upgrade bool) error {
	s := spinner.New(flagDebug)
	s.WithIndicator(fmt.Sprintf("Installing %s...\n", ext))
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
			return fmt.Errorf("has already been installed outside of pgxman, run with `--overwrite` to overwrite it")
		}

		return fmt.Errorf("failed to install, run with `--debug` to see the full error: %w", err)
	}

	s.Start()
	if err := f(ctx, ext); err != nil {
		err = handleErr(err)
		s.WithDone(fmt.Sprintf("[%s] %s: %s\n", errorMark, ext, err))
		return err
	}

	s.WithDone(fmt.Sprintf("[%s] %s: https://pgx.sh/%s\n", successMark, ext, ext.Name))

	return nil
}

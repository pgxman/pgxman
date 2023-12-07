package pgxman

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/pg"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/pgxman/pgxman/internal/registry"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	flagInstallOrUpgradeYes       bool
	flagInstallOrUpgradePGVersion string
	flagInstallOrUpgradeOverwrite bool
)

func newInstallOrUpgradeCmd(upgrade bool) *cobra.Command {
	action := "install"
	if upgrade {
		action = "upgrade"
	}

	var defPGVer string
	pgVer, err := pg.DetectVersion(context.Background())
	if err == nil {
		defPGVer = string(pgVer)
	}

	exampleTmpl := `  # {{ title .Action }} the latest pgvector for the installed PostgreSQL.
  # PostgreSQL version is detected from pg_config if it exists,
  # Otherwise, the latest supported PostgreSQL version is used.
  pgxman {{ .Action }} pgvector

  # {{ title .Action }} the latest pgvector for PostgreSQL {{ .PGVer }}
  pgxman {{ .Action }} pgvector --pg {{ .PGVer }}

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL {{ .PGVer }}
  pgxman {{ .Action }} pgvector=0.5.0 --pg {{ .PGVer }}

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL {{ .PGVer }} with sudo
  pgxman {{ .Action }} pgvector=0.5.0 --pg {{ .PGVer }} --sudo

  # {{ title .Action }} pgvector 0.5.0 and postgis 3.3.3 for PostgreSQL {{ .PGVer }}
  pgxman {{ .Action }} pgvector=0.5.0 postgis=3.3.3 --pg {{ .PGVer }}

  # {{ title .Action }} from a local Debian package
  pgxman {{ .Action }} /PATH_TO/postgresql-15-pgxman-pgvector_0.5.0_arm64.deb`

	type data struct {
		Action string
		PGVer  string
	}

	c := cases.Title(language.AmericanEnglish)
	funcMap := template.FuncMap{
		"title": c.String,
	}

	buf := bytes.NewBuffer(nil)
	if err := template.Must(
		template.New("").Funcs(funcMap).Parse(exampleTmpl),
	).Execute(
		buf,
		data{
			Action: action,
			PGVer:  string(pgxman.DefaultPGVersion),
		},
	); err != nil {
		// impossible
		panic(err.Error())
	}

	cmd := &cobra.Command{
		Use:   action,
		Short: c.String(action) + " PostgreSQL extensions",
		Long: c.String(action) + ` PostgreSQL extensions from commandline arguments. The argument
format is NAME=VERSION. The PostgreSQL version is detected from pg_config
if it exists, or can be specified with the --pg flag.`,
		Example: buf.String(),
		RunE:    runInstallOrUpgrade(upgrade),
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.PersistentFlags().BoolVarP(&flagInstallOrUpgradeYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)
	cmd.PersistentFlags().StringVar(&flagInstallOrUpgradePGVersion, "pg", defPGVer, fmt.Sprintf("%s the extension for the PostgreSQL version. It detects the version by pg_config if it exists. Supported values are %s.", c.String(action), strings.Join(supportedPGVersions(), ", ")))
	cmd.PersistentFlags().BoolVar(&flagInstallOrUpgradeOverwrite, "overwrite", false, "Overwrite the existing extension if it is installed outside of pgxman.")

	return cmd
}

func runInstallOrUpgrade(upgrade bool) func(c *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		i, err := plugin.GetInstaller()
		if err != nil {
			return errorsx.Pretty(err)
		}

		if len(args) == 0 {
			return fmt.Errorf("need at least one extension")
		}

		pgVer := pgxman.PGVersion(flagInstallOrUpgradePGVersion)
		if err := checkPGVerExists(cmd.Context(), pgVer); err != nil {
			return err
		}

		client, err := newReigstryClient()
		if err != nil {
			return err
		}

		p := NewArgsParser(
			client,
			DefaultPlatformDetector,
			pgVer,
			flagInstallOrUpgradeOverwrite,
		)
		exts, err := p.Parse(cmd.Context(), args)
		if err != nil {
			return err
		}

		if !flagInstallOrUpgradeYes {
			checkFunc := i.PreInstallCheck
			if upgrade {
				checkFunc = i.PreUpgradeCheck
			}

			if err := checkFunc(cmd.Context(), exts, pgxman.NewStdIO()); err != nil {
				return err
			}
		}

		action := "Installing"
		if upgrade {
			action = "Upgrading"
		}

		logger := log.NewTextLogger()
		fmt.Printf("%s extensions for PostgreSQL %s...\n", action, pgVer)
		for _, ext := range exts {
			if err := installOrUpgrade(cmd.Context(), i, ext, upgrade); err != nil {
				logger.Debug("failed to install extension", "error", err, "extension", ext)
				os.Exit(1)
			}
		}

		if upgrade {
			fmt.Println(`After restarting PostgreSQL, update extensions in each database by running in the psql shell:

    ALTER EXTENSION name UPDATE`)
		}

		return nil
	}
}

type errInvalidExtensionFormat struct {
	Arg string
}

func (e errInvalidExtensionFormat) Error() string {
	return fmt.Sprintf("invalid extension format: %q. The format is NAME=VERSION...", e.Arg)
}

type errInvalidPGVersion struct {
	Version pgxman.PGVersion
}

func (e errInvalidPGVersion) Error() string {
	msg := "could not detect an installation of Postgres"
	if e.Version != pgxman.PGVersionUnknown {
		msg = fmt.Sprintf("could not detect an installation of Postgres %s", e.Version)
	}

	return fmt.Sprintf("%s. For information on installing Postgres, see: https://docs.pgxman.com/installing_postgres.", msg)
}

func supportedPGVersions() []string {
	var pgVers []string
	for _, v := range pgxman.SupportedPGVersions {
		pgVers = append(pgVers, string(v))
	}

	return pgVers
}

func checkPGVerExists(ctx context.Context, pgVer pgxman.PGVersion) error {
	if pgVer == pgxman.PGVersionUnknown || !pg.VersionExists(ctx, pgVer) {
		return errInvalidPGVersion{Version: pgVer}
	}

	return nil
}

func newReigstryClient() (registry.Client, error) {
	return registry.NewClient(flagRegistryURL)
}

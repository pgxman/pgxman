package pgxman

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/pg"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	flagInstallerYes       bool
	flagInstallerSudo      bool
	flagInstallerPGVersion string
)

func newInstallOrUpgradeCmd(upgrade bool) *cobra.Command {
	action := "install"
	if upgrade {
		action = "upgrade"
	}

	exampleTmpl := `  # {{ title .Action }} the latest pgvector for the installed PostgreSQL.
  # PostgreSQL version is detected from pg_config if it exists,
  # Otherwise, the latest supported PostgreSQL version is used.
  pgxman {{ .Action }} pgvector

  # {{ title .Action }} the latest pgvector for PostgreSQL 14
  pgxman {{ .Action }} pgvector --pg 14

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL 14
  pgxman {{ .Action }} pgvector=0.5.0 --pg 14

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL 14 with sudo
  pgxman {{ .Action }} pgvector=0.5.0 --pg 14 --sudo

  # {{ title .Action }} pgvector 0.5.0 and postgis 3.3.3 for PostgreSQL 14
  pgxman {{ .Action }} pgvector=0.5.0 postgis=3.3.3 --pg 14

  # {{ title .Action }} from a local Debian package
  pgxman {{ .Action }} /PATH_TO/postgresql-15-pgxman-pgvector_0.5.0_arm64.deb`

	type data struct {
		Action string
	}

	c := cases.Title(language.AmericanEnglish)
	funcMap := template.FuncMap{
		"title": c.String,
	}

	buf := bytes.NewBuffer(nil)
	if err := template.Must(template.New("").Funcs(funcMap).Parse(exampleTmpl)).Execute(buf, data{Action: action}); err != nil {
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

	cmd.PersistentFlags().BoolVar(&flagInstallerSudo, "sudo", os.Getenv("PGXMAN_SUDO") != "", "Run the underlaying package manager command with sudo.")
	cmd.PersistentFlags().BoolVarP(&flagInstallerYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)
	cmd.PersistentFlags().StringVar(&flagInstallerPGVersion, "pg", string(pg.DefaultVersion(context.Background())), "Install the extension for the PostgreSQL version identified by pg_config, if it exists.")

	return cmd
}

func runInstallOrUpgrade(upgrade bool) func(c *cobra.Command, args []string) error {
	return func(c *cobra.Command, args []string) error {
		i, err := plugin.GetInstaller()
		if err != nil {
			return errorsx.Pretty(err)
		}

		if len(args) == 0 {
			return fmt.Errorf("need at least one extension")
		}

		f, err := parseFromCommandLine(args, pgxman.PGVersion(flagInstallerPGVersion))
		if err != nil {
			return err
		}

		if upgrade {
			if err := i.Upgrade(
				c.Context(),
				[]pgxman.PGXManfile{*f},
				pgxman.InstallOptWithIgnorePrompt(flagInstallerYes),
				pgxman.InstallOptWithSudo(flagInstallerSudo),
			); err != nil {
				return err
			}

			fmt.Println(`After restarting PostgreSQL, update extensions in each database by running:

  ALTER EXTENSION name UPDATE`)

			return nil
		}

		return i.Install(
			c.Context(),
			[]pgxman.PGXManfile{*f},
			pgxman.InstallOptWithIgnorePrompt(flagInstallerYes),
			pgxman.InstallOptWithSudo(flagInstallerSudo),
		)
	}
}

type errInvalidExtensionFormat struct {
	Arg string
}

func (e errInvalidExtensionFormat) Error() string {
	return fmt.Sprintf("invalid extension format: %q. The format is NAME=VERSION@PGVERSION1,PGVERSION2...", e.Arg)
}

var (
	extRegexp = regexp.MustCompile(`^([^=@\s]+)(?:=([^@]*))?$`)
)

func parseFromCommandLine(args []string, pgVer pgxman.PGVersion) (*pgxman.PGXManfile, error) {
	if !slices.Contains(pgxman.SupportedPGVersions, pgVer) {
		return nil, fmt.Errorf("unsupported PostgreSQL version: %q", pgVer)
	}

	var exts []pgxman.InstallExtension
	for _, arg := range args {
		ext, err := parseInstallExtension(arg)
		if err != nil {
			return nil, err
		}

		exts = append(exts, *ext)
	}

	return &pgxman.PGXManfile{
		APIVersion: pgxman.DefaultPGXManfileAPIVersion,
		Extensions: exts,
		PGVersions: []pgxman.PGVersion{pgxman.PGVersion(pgVer)},
	}, nil
}

func parseInstallExtension(arg string) (*pgxman.InstallExtension, error) {
	// install from local file
	if _, err := os.Stat(arg); err == nil {
		path, err := filepath.Abs(arg)
		if err != nil {
			return nil, err
		}

		return &pgxman.InstallExtension{
			Path: path,
		}, nil
	}

	// install from apt
	if extRegexp.MatchString(arg) {
		var (
			match   = extRegexp.FindStringSubmatch(arg)
			name    = match[1]
			version = match[2]
		)

		return &pgxman.InstallExtension{
			Name:    name,
			Version: version,
		}, nil
	}

	return nil, errInvalidExtensionFormat{Arg: arg}
}

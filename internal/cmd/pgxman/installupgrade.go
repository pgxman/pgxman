package pgxman

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/pg"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/pgxman/pgxman/internal/registry"
	"github.com/pgxman/pgxman/oapi"
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

		p := &ArgsParser{
			Overwrite: flagInstallOrUpgradeOverwrite,
			PGVer:     pgVer,
			Logger:    log.NewTextLogger(),
		}
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

		fmt.Printf("%s extensions for PostgreSQL %s...\n", action, pgVer)
		for _, ext := range exts {
			if err := installOrUpgrade(cmd.Context(), i, ext, upgrade); err != nil {
				// Error message is already shown in spinner
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

type ArgsParser struct {
	PGVer     pgxman.PGVersion
	Overwrite bool
	Logger    *log.Logger
}

func (p *ArgsParser) Parse(ctx context.Context, args []string) ([]pgxman.InstallExtension, error) {
	if err := pgxman.ValidatePGVersion(p.PGVer); err != nil {
		return nil, err
	}

	var exts []pgxman.InstallExtension
	for _, arg := range args {
		ext, err := parseInstallExtension(arg)
		if err != nil {
			return nil, err
		}
		ext.Overwrite = p.Overwrite

		exts = append(exts, pgxman.InstallExtension{
			PackExtension: *ext,
			PGVersion:     p.PGVer,
		})
	}

	locker, err := NewExtensionLocker(p.Logger)
	if err != nil {
		return nil, err
	}

	return locker.Lock(ctx, exts)
}

func NewExtensionLocker(logger *log.Logger) (*ExtensionLocker, error) {
	c, err := registry.NewClient(flagRegistryURL)
	if err != nil {
		return nil, err
	}

	return &ExtensionLocker{
		Client: c,
		Logger: logger,
	}, nil
}

type ExtensionLocker struct {
	Client *registry.Client
	Logger *log.Logger
}

func (l *ExtensionLocker) Lock(ctx context.Context, exts []pgxman.InstallExtension) ([]pgxman.InstallExtension, error) {
	p, err := pgxman.DetectPlatform()
	if err != nil {
		return nil, fmt.Errorf("detect platform: %s", err)
	}

	var result []pgxman.InstallExtension
	for _, ext := range exts {
		if ext.Name != "" {
			installableExt, err := l.Client.GetExtension(ctx, ext.Name)
			if err != nil {
				if errors.Is(err, registry.ErrExtensionNotFound) {
					return nil, fmt.Errorf("extension %q not found", ext.Name)
				}
			}

			// if version is not specified, use the latest version
			if ext.Version == "" || ext.Version == "latest" {
				ext.Version = installableExt.Version
			}

			if installableExt.Version != ext.Version {
				// TODO(owenthereal): validate old version when api is ready
				l.Logger.Debug("extension version does not match the latest", "extension", ext.Name, "version", ext.Version, "latest", installableExt.Version)
			}

			platform, err := installableExt.GetPlatform(p)
			if err != nil {
				return nil, err
			}

			if !slices.Contains(platform.PgVersions, convertPGVersion(ext.PGVersion)) {
				return nil, fmt.Errorf("%s %s is incompatible with PostgreSQL %s", ext.Name, ext.Version, ext.PGVersion)
			}
		}

		result = append(result, ext)
	}

	return result, nil
}

var (
	extRegexp = regexp.MustCompile(`^([^=@\s]+)(?:=([^@]*))?$`)
)

func parseInstallExtension(arg string) (*pgxman.PackExtension, error) {
	// install from local file
	if _, err := os.Stat(arg); err == nil {
		path, err := filepath.Abs(arg)
		if err != nil {
			return nil, err
		}

		return &pgxman.PackExtension{
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

		return &pgxman.PackExtension{
			Name:    name,
			Version: version,
		}, nil
	}

	return nil, errInvalidExtensionFormat{Arg: arg}
}

func supportedPGVersions() []string {
	var pgVers []string
	for _, v := range pgxman.SupportedPGVersions {
		pgVers = append(pgVers, string(v))
	}

	return pgVers
}

func installExts(b pgxman.Pack) []pgxman.InstallExtension {
	var installExts []pgxman.InstallExtension
	for _, ext := range b.Extensions {
		installExts = append(installExts, pgxman.InstallExtension{
			PackExtension: ext,
			PGVersion:     b.Postgres.Version,
		})
	}

	return installExts
}

func checkPGVerExists(ctx context.Context, pgVer pgxman.PGVersion) error {
	if pgVer == pgxman.PGVersionUnknown || !pg.VersionExists(ctx, pgVer) {
		return errInvalidPGVersion{Version: pgVer}
	}

	return nil
}

func convertPGVersion(pgVer pgxman.PGVersion) oapi.PgVersion {
	switch pgVer {
	case pgxman.PGVersion13:
		return oapi.Pg13
	case pgxman.PGVersion14:
		return oapi.Pg14
	case pgxman.PGVersion15:
		return oapi.Pg15
	case pgxman.PGVersion16:
		return oapi.Pg16
	default:
		panic(fmt.Sprintf("invalid pg version: %s", pgVer))
	}
}

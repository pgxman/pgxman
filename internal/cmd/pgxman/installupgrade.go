package pgxman

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/log"
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

	defPGVer := string(pg.DetectVersion(context.Background()))

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

	cmd.PersistentFlags().BoolVar(&flagInstallerSudo, "sudo", os.Getenv("PGXMAN_SUDO") != "", "Run the underlaying package manager command with sudo.")
	cmd.PersistentFlags().BoolVarP(&flagInstallerYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)
	cmd.PersistentFlags().StringVar(&flagInstallerPGVersion, "pg", defPGVer, fmt.Sprintf("Install the extension for the PostgreSQL version. It detects the version by pg_config if it exists. Supported values are %s.", strings.Join(supportedPGVersions(), ", ")))

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

		p := &ArgsParser{
			PGVer:  pgxman.PGVersion(flagInstallerPGVersion),
			Logger: log.NewTextLogger(),
		}
		f, err := p.Parse(args)
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
	return fmt.Sprintf("invalid extension format: %q. The format is NAME=VERSION...", e.Arg)
}

type ArgsParser struct {
	PGVer  pgxman.PGVersion
	Logger *log.Logger
}

func (p *ArgsParser) Parse(args []string) (*pgxman.PGXManfile, error) {
	var exts []pgxman.InstallExtension
	for _, arg := range args {
		ext, err := parseInstallExtension(arg)
		if err != nil {
			return nil, err
		}

		exts = append(exts, *ext)
	}

	f := &pgxman.PGXManfile{
		APIVersion: pgxman.DefaultPGXManfileAPIVersion,
		Extensions: exts,
		PGVersions: []pgxman.PGVersion{p.PGVer},
	}
	if err := LockPGXManfile(f, p.Logger); err != nil {
		return nil, err
	}

	return f, nil
}

func LockPGXManfile(f *pgxman.PGXManfile, logger *log.Logger) error {
	if err := f.Validate(); err != nil {
		return err
	}

	installableExts, err := buildkit.Extensions()
	if err != nil {
		return fmt.Errorf("fetch installable extensions: %w", err)
	}

	var exts []pgxman.InstallExtension
	for _, ext := range f.Extensions {
		if ext.Name != "" {
			installableExt, ok := installableExts[ext.Name]
			if !ok {
				return fmt.Errorf("extension %q not found", ext.Name)
			}

			// if version is not specified, use the latest version
			if ext.Version == "" || ext.Version == "latest" {
				ext.Version = installableExt.Version
			}

			if installableExt.Version != ext.Version {
				// TODO(owenthereal): validate old version when api is ready
				logger.Debug("extension version does not match the latest", "extension", ext.Name, "version", ext.Version, "latest", installableExt.Version)
			}
		}

		exts = append(exts, ext)
	}

	f.Extensions = exts

	return nil
}

var (
	extRegexp = regexp.MustCompile(`^([^=@\s]+)(?:=([^@]*))?$`)
)

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

func supportedPGVersions() []string {
	var pgVers []string
	for _, v := range pgxman.SupportedPGVersions {
		pgVers = append(pgVers, string(v))
	}

	return pgVers
}

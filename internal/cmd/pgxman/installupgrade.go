package pgxman

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/briandowns/spinner"
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
	flagInstallOrUpgradeYes       bool
	flagInstallOrUpgradePGVersion string
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
	cmd.PersistentFlags().StringVar(&flagInstallOrUpgradePGVersion, "pg", defPGVer, fmt.Sprintf("Install the extension for the PostgreSQL version. It detects the version by pg_config if it exists. Supported values are %s.", strings.Join(supportedPGVersions(), ", ")))

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
		if err := validatePGVer(cmd.Context(), pgVer); err != nil {
			return err
		}

		p := &ArgsParser{
			PGVer:  pgVer,
			Logger: log.NewTextLogger(),
		}
		f, err := p.Parse(cmd.Context(), args)
		if err != nil {
			return err
		}

		s := newSpinner()
		defer s.Stop()

		exts := extNames(f.Extensions)
		var (
			action     = "Installing"
			actionVerb = "install"
		)
		if upgrade {
			action = "Upgrading"
			actionVerb = "upgrade"
		}

		s.Suffix = fmt.Sprintf(" %s %s for PostgreSQL %s...\n", action, exts, pgVer)

		var opts []pgxman.InstallerOptionsFunc
		if flagInstallOrUpgradeYes {
			s.Start()
		} else {
			opts = append(opts, pgxman.WithBeforeRunHook(func(debPkgs []string, sources []string) error {
				if err := promptInstallOrUpgrade(debPkgs, sources, upgrade); err != nil {
					return err
				}

				s.Start()
				return nil
			}))
		}

		handleErr := func(err error) error {
			if errors.Is(err, pgxman.ErrRootAccessRequired) {
				return fmt.Errorf("must run command as root: sudo %s", strings.Join(os.Args, " "))
			}

			return fmt.Errorf("failed to %s %s, run with `--debug` to see the full error: %w", actionVerb, exts, err)
		}

		if upgrade {
			if err := i.Upgrade(
				cmd.Context(),
				*f,
				opts...,
			); err != nil {
				return handleErr(err)
			}

			s.FinalMSG = fmt.Sprintf(`%s
After restarting PostgreSQL, update extensions in each database by running in the psql shell:

    ALTER EXTENSION name UPDATE
`, extOutput(f))
			return nil
		}

		if err := i.Install(
			cmd.Context(),
			*f,
			opts...,
		); err != nil {
			return handleErr(err)
		}

		s.FinalMSG = extOutput(f)
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
	PGVer  pgxman.PGVersion
	Logger *log.Logger
}

func (p *ArgsParser) Parse(ctx context.Context, args []string) (*pgxman.PGXManfile, error) {
	if err := pgxman.ValidatePGVersion(p.PGVer); err != nil {
		return nil, err
	}
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
		Postgres: pgxman.Postgres{
			Version: p.PGVer,
		},
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

func extOutput(f *pgxman.PGXManfile) string {
	var lines []string
	for _, ext := range f.Extensions {
		lines = append(lines, fmt.Sprintf("[%s] %s: %s", successMark, extName(ext), extLink(ext)))
	}

	return strings.Join(lines, "\n") + "\n"
}

func extNames(exts []pgxman.InstallExtension) string {
	var names []string
	for _, ext := range exts {
		names = append(names, extName(ext))
	}

	return strings.Join(names, ", ")
}

func extName(ext pgxman.InstallExtension) string {
	if ext.Name != "" {
		return ext.Name
	}

	return ext.Path
}

func extLink(ext pgxman.InstallExtension) string {
	return fmt.Sprintf("https://pgx.sh/%s", ext.Name)
}

func promptInstallOrUpgrade(debPkgs []string, sources []string, upgrade bool) error {
	var (
		action   = "installed"
		abortMsg = "installation aborted"
	)
	if upgrade {
		action = "upgraded"
		abortMsg = "upgrade aborted"
	}

	out := []string{
		fmt.Sprintf("The following Debian packages will be %s:", action),
	}
	for _, debPkg := range debPkgs {
		out = append(out, "  "+debPkg)
	}

	if len(sources) > 0 {
		out = append(out, "The following Apt repositories will be added or updated:")
		for _, source := range sources {
			out = append(out, "  "+source)
		}
	}

	out = append(out, "Do you want to continue? [Y/n] ")
	fmt.Print(strings.Join(out, "\n"))

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		switch strings.ToLower(scanner.Text()) {
		case "y", "yes", "":
			return nil
		default:
			return fmt.Errorf(abortMsg)
		}
	}

	return nil
}

func validatePGVer(ctx context.Context, pgVer pgxman.PGVersion) error {
	if pgVer == pgxman.PGVersionUnknown || !pg.VersionExists(ctx, pgVer) {
		return errInvalidPGVersion{Version: pgVer}
	}

	return nil
}

func newSpinner() *spinner.Spinner {
	return spinner.New(spinner.CharSets[9], 100*time.Millisecond)
}

package pgxman

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	flagInstallerYes  bool
	flagInstallerSudo bool
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
  pgxman {{ .Action }} pgvector@14

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL 14
  pgxman {{ .Action }} pgvector=0.5.0@14

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL 14 with sudo
  pgxman {{ .Action }} pgvector=0.5.0@14 --sudo

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL 14 & 15
  pgxman {{ .Action }} pgvector=0.5.0@14,15

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL 14 & 15, and postgis 3.3.3 for PostgreSQL 14
  pgxman {{ .Action }} pgvector=0.5.0@14,15 postgis=3.3.3@14

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
format is NAME=VERSION@PGVERSIONS where PGVERSIONS is a comma separated
list of PostgreSQL versions.`,
		Example: buf.String(),
		RunE:    runInstallOrUpgrade(upgrade),
		Args:    cobra.MinimumNArgs(1),
	}

	cmd.PersistentFlags().BoolVar(&flagInstallerSudo, "sudo", os.Getenv("PGXMAN_SUDO") != "", "Run the underlaying package manager command with sudo.")
	cmd.PersistentFlags().BoolVarP(&flagInstallerYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)

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

		var result []pgxman.PGXManfile
		for _, arg := range args {
			exts, err := parseInstallExtensions(arg)
			if err != nil {
				return err
			}

			result = append(result, *exts)
		}

		if upgrade {
			if err := i.Upgrade(
				c.Context(),
				result,
				pgxman.InstallOptWithIgnorePrompt(flagInstallerYes),
				pgxman.InstallOptWithSudo(flagInstallerSudo),
			); err != nil {
				return err
			}

			// print warning
		}

		return i.Install(
			c.Context(),
			result,
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
	extRegexp = regexp.MustCompile(`^([^=@\s]+)(?:=([^@]*))?(?:@(\S+))?$`)
)

func parseInstallExtensions(arg string) (*pgxman.PGXManfile, error) {
	// install from apt
	if extRegexp.MatchString(arg) {
		match := extRegexp.FindStringSubmatch(arg)
		var (
			name       = match[1]
			version    = match[2]
			pgversions = strings.Split(match[3], ",")
		)

		if len(pgversions) == 0 {
			return nil, errInvalidExtensionFormat{Arg: arg}
		}

		if len(pgversions) == 1 && pgversions[0] == "" {
			pgversions = []string{string(pgxman.PGVersionUnknown)}
		}

		var (
			pgvers []pgxman.PGVersion
			exts   = []pgxman.InstallExtension{
				{
					Name:    name,
					Version: version,
				},
			}
		)
		for _, pgversion := range pgversions {
			pgvers = append(pgvers, pgxman.PGVersion(pgversion))
		}

		return &pgxman.PGXManfile{
			APIVersion: pgxman.DefaultPGXManfileAPIVersion,
			Extensions: exts,
			PGVersions: pgvers,
		}, nil
	}

	// install from local file
	if _, err := os.Stat(arg); err == nil {
		path, err := filepath.Abs(arg)
		if err != nil {
			return nil, err
		}

		return &pgxman.PGXManfile{
			APIVersion: pgxman.DefaultPGXManfileAPIVersion,
			Extensions: []pgxman.InstallExtension{
				{
					Path: path,
				},
			},
			PGVersions: pgxman.SupportedPGVersions,
		}, nil
	}

	return nil, errInvalidExtensionFormat{Arg: arg}
}

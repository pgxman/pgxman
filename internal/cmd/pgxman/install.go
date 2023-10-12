package pgxman

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/errorsx"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	flagInstallPGXManfile string
	flagInstallYes        bool
	flagInstallSudo       bool
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install PostgreSQL extensions",
		Long: `Install PostgreSQL extensions from a pgxman.yaml file or from commandline arguments. To install from arguments, the
format is NAME=VERSION@PGVERSIONS where PGVERSIONS is a comma separated list of PostgreSQL versions.`,
		Example: `  # Install extensions from the pgxman.yaml file in the current directory
  pgxman install

  # Install extensions by surpressing prompts
  pgxman install -y

  # Install extensions from the pgxman.yaml in a specific directory
  pgxman install -f /PATH_TO/pgxman.yaml

  # Install extensions from STDIN with the pgxman.yaml format
  cat <<EOF | pgxman install -f -
    apiVersion: v1
    extensions:
      - name: "pgvector"
        version: "0.5.0"
      - path: "/local/path/to/extension"
    pgVersions:
      - "14"
      - "15"
  EOF

  # Install the latest pgvector for the installed PostgreSQL.
  # PostgreSQL version is detected from pg_config if it exists,
  # Otherwise, the latest supported PostgreSQL version is used.
  pgxman install pgvector

  # Install the latest pgvector for PostgreSQL 14
  pgxman install pgvector@14

  # Install pgvector 0.5.0 for PostgreSQL 14
  pgxman install pgvector=0.5.0@14

  # Install pgvector 0.5.0 for PostgreSQL 14 with sudo
  pgxman install pgvector=0.5.0@14 --sudo

  # Install pgvector 0.5.0 for PostgreSQL 14 & 15
  pgxman install pgvector=0.5.0@14,15

  # Install pgvector 0.5.0 for PostgreSQL 14 & 15, and postgis 3.3.3 for PostgreSQL 14
  pgxman install pgvector=0.5.0@14,15 postgis=3.3.3@14

  # Install from a local Debian package
  pgxman install /PATH_TO/postgresql-15-pgxman-pgvector_0.5.0_arm64.deb`,
		RunE: runInstall,
	}

	cmd.PersistentFlags().StringVarP(&flagInstallPGXManfile, "file", "f", "", "The pgxman.yaml file to use. Defaults to pgxman.yaml in the current directory.")
	cmd.PersistentFlags().BoolVar(&flagInstallSudo, "sudo", os.Getenv("PGXMAN_SUDO") != "", "Run the underlaying package manager command with sudo.")
	cmd.PersistentFlags().BoolVarP(&flagInstallYes, "yes", "y", false, `Automatic yes to prompts and run install non-interactively.`)

	return cmd
}

func runInstall(c *cobra.Command, args []string) error {
	i, err := plugin.GetInstaller()
	if err != nil {
		return errorsx.Pretty(err)
	}

	var result []pgxman.PGXManfile

	if len(args) == 0 {
		if flagInstallPGXManfile == "" {
			pwd, err := os.Getwd()
			if err != nil {
				return err
			}

			flagInstallPGXManfile = filepath.Join(pwd, "pgxman.yaml")
		}

		pgxmf, err := pgxman.ReadPGXManfile(flagInstallPGXManfile)
		if err != nil {
			return err
		}

		result = append(result, *pgxmf)
	} else {
		for _, arg := range args {
			exts, err := parseInstallExtensions(arg)
			if err != nil {
				return err
			}

			result = append(result, *exts)
		}
	}

	if err := i.Install(
		c.Context(),
		result,
		pgxman.InstallOptWithIgnorePrompt(flagInstallYes),
		pgxman.InstallOptWithSudo(flagInstallSudo),
	); err != nil {

		return err
	}

	return nil
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

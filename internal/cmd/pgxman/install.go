package pgxman

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/plugin"
	"github.com/spf13/cobra"
)

var (
	flagInstallPGXManfile string
	flagInstallYes        bool
)

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install PostgreSQL extensions",
		Long: `Install PostgreSQL extensions from a pgxman.yaml file or from commandline arguments. To install from arguments, the
format is NAME=VERSION@PGVERSIONS where PGVERSIONS is a comma separated list of PostgreSQL versions.`,
		Example: `  # Install extensions from the pgxman.yaml file in the current directory
  pgxman install

  # Install extensions from the pgxman.yaml in a specific directory
  pgxman install -f /PATH_TO/pgxman.yaml

  # Install extensions from STDIN with the pgxman.yaml format
  cat <<EOF | pgxman install -f -
    apiVersion: v1
    extensions:
      - name: "pgvector"
        version: "0.4.4"
      - path: "/local/path/to/extension"
    pgVersions:
      - "14"
      - "15"
  EOF

  # Install pgvector 0.4.4 for PostgreSQL 14
  pgxman install pgvector=0.4.4@14

  # Install pgvector 0.4.4 for PostgreSQL 14 & 15
  pgxman install pgvector=0.4.4@14,15

  # Install pgvector 0.4.4 for PostgreSQL 14 & 15, and postgis 3.3.3 for PostgreSQL 14
  pgxman install pgvector=0.4.4@14,15 postgis=3.3.3@14

  # Install from a local Debian package
  pgxman install /PATH_TO/postgresql-15-pgxman-pgvector_0.4.4_arm64.deb`,
		RunE: runInstall,
	}

	cmd.PersistentFlags().StringVarP(&flagInstallPGXManfile, "file", "f", "", "The pgxman.yaml file to use. Defaults to pgxman.yaml in the current directory.")
	cmd.PersistentFlags().BoolVarP(&flagInstallYes, "yes", "y", false, `Automatic yes to prompts; assume "yes" as answer to all prompts and run non-interactively.`)

	return cmd
}

func runInstall(c *cobra.Command, args []string) error {
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

	i, err := plugin.GetInstaller()
	if err != nil {
		return err
	}

	if err := i.Install(
		c.Context(),
		result,
		pgxman.InstallOptWithIgnorePrompt(flagInstallYes),
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
	extRegexp = regexp.MustCompile(`^(.+)=(.+)@(.+)$`)
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

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

func newInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a PostgreSQL extension",
		Long: `Install a PostgreSQL extension from registry or from local paths. To install from the registry, the
format is NAME=VERSION@PGVERSIONS where PGVERSIONS is a comma separated list of PostgreSQL versions.`,
		Example: `  # Install pgvector 0.4.4 for PostgreSQL 14
  pgxman install pgvector=0.4.4@14

  # Install pgvector 0.4.4 for PostgreSQL 14 & 15
  pgxman install pgvector=0.4.4@14,15

  # Install pgvector 0.4.4 for PostgreSQL 14 & 15, and postgis 3.3.3 for PostgreSQL 14
  pgxman install pgvector=0.4.4@14,15 postgis=3.3.3@14

  # Install from a local Debian package
  pgxman install /PATH_TO/postgresql-15-pgxman-pgvector_0.4.4_arm64.deb`,
		RunE: runInstall,
	}

	return cmd
}

func runInstall(c *cobra.Command, args []string) error {
	var result []pgxman.InstallExtension
	for _, arg := range args {
		exts, err := parseInstallExtensions(arg)
		if err != nil {
			return err
		}

		result = append(result, exts...)
	}

	i, err := plugin.GetInstaller()
	if err != nil {
		return err
	}

	return i.Install(c.Context(), result)
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

func parseInstallExtensions(arg string) ([]pgxman.InstallExtension, error) {
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

		var exts []pgxman.InstallExtension
		for _, pgversion := range pgversions {
			exts = append(exts, pgxman.InstallExtension{
				Name:      name,
				Version:   version,
				PGVersion: pgxman.PGVersion(pgversion),
			})
		}

		return exts, nil
	}

	// install from local file
	if _, err := os.Stat(arg); err == nil {
		path, err := filepath.Abs(arg)
		if err != nil {
			return nil, err
		}

		return []pgxman.InstallExtension{
			{
				Path: path,
			},
		}, nil
	}

	return nil, errInvalidExtensionFormat{Arg: arg}
}

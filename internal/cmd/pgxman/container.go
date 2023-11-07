package pgxman

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/container"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	flagContainerInstallRunnerImage string
)

func newContainerCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "container",
		Short:   "Run virtualized playground in a container",
		Aliases: []string{"c"},
	}

	root.AddCommand(newContainerInstallCmd())
	root.AddCommand(newContainerTeardownCmd())

	return root
}

func newContainerInstallCmd() *cobra.Command {
	exampleTmpl := `  # {{ title .Action }} the latest pgvector in a container.
  pgxman container {{ .Action }} pgvector

  # {{ title .Action }} the latest pgvector for PostgreSQL {{ .PGVer }} in a container.
  pgxman container {{ .Action }} pgvector --pg {{ .PGVer }}

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL {{ .PGVer }} in a container.
  pgxman container {{ .Action }} pgvector=0.5.0 --pg {{ .PGVer }}

  # {{ title .Action }} pgvector 0.5.0 and postgis 3.3.3 for PostgreSQL {{ .PGVer }} in a container
  pgxman container {{ .Action }} pgvector=0.5.0 postgis=3.3.3 --pg {{ .PGVer }}

  # {{ title .Action }} a local Debian package in a container
  pgxman container {{ .Action }} /PATH_TO/postgresql-15-pgxman-pgvector_0.5.0_arm64.deb`

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
			Action: "install",
			PGVer:  string(pgxman.DefaultPGVersion),
		},
	); err != nil {
		// impossible
		panic(err.Error())
	}

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install PostgreSQL extensions in a container",
		Long: `Start a container with the specified PostgreSQL version and install
PostgreSQL extension from commandline arguments. The argument format
is NAME=VERSION.`,
		Example: buf.String(),
		RunE:    runContainerInstall,
		Args:    cobra.MinimumNArgs(1),
	}

	defPGVer := string(pgxman.DefaultPGVersion)

	cmd.PersistentFlags().StringVar(&flagInstallerPGVersion, "pg", defPGVer, fmt.Sprintf("Install the extension for the PostgreSQL version. Supported values are %s.", strings.Join(supportedPGVersions(), ", ")))
	cmd.PersistentFlags().StringVar(&flagContainerInstallRunnerImage, "runner-image", "", "Override the default runner image")

	return cmd
}

func runContainerInstall(cmd *cobra.Command, args []string) error {
	p := &ArgsParser{
		PGVer:  pgxman.PGVersion(flagInstallerPGVersion),
		Logger: log.NewTextLogger(),
	}
	f, err := p.Parse(args)
	if err != nil {
		return err
	}

	var exts []string
	for _, ext := range f.Extensions {
		if ext.Name != "" {
			exts = append(exts, ext.Name)
		} else if ext.Path != "" {
			exts = append(exts, ext.Path)
		}
	}

	info, err := container.NewContainer(
		container.WithRunnerImage(flagContainerInstallRunnerImage),
	).Install(cmd.Context(), f)
	if err != nil {
		return err
	}

	fmt.Printf(`%s installed successfully.

pgxman is running in a Docker container. To connect, run:

    $ psql postgres://%s:%s@127.0.0.1:%s/%s

To tear down the container, run:

    $ pgxman container teardown %s

For more information on the docker environment, please see: https://docs.pgxman.com/container.
`,
		strings.Join(exts, ", "),
		info.PGUser,
		info.PGPassword,
		info.Port,
		info.PGDatabase,
		info.PGVersion,
	)

	return nil
}

func newContainerTeardownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teardown",
		Short: "Tear down a playground container",
		Long:  `Tear down a playground container and purge all data.`,
		Example: ` # Tear down the PostgreSQL 15 playgrond container.
pgxman container teardown 15

# Tear down the PostgreSQL 15 & 16 containers.
pgxman container teardown 15 16
`,
		RunE: runContainerTeardown,
		Args: cobra.MinimumNArgs(1),
	}

	return cmd
}

func runContainerTeardown(cmd *cobra.Command, args []string) error {
	c := container.NewContainer()
	for _, arg := range args {
		pgVer := pgxman.PGVersion(arg)

		if !pgxman.IsSupportedPGVersion(pgVer) {
			return fmt.Errorf("unsupported PostgreSQL version: %s", pgVer)
		}

		if err := c.Teardown(cmd.Context(), pgVer); err != nil {
			return err
		}
	}

	return nil
}

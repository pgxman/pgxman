package pgxman

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
	"github.com/pgxman/pgxman/internal/container"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	flagContainerInstallRunnerImage string
	flagContainerInstallPGVersion   string
)

func newContainerCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "container",
		Short:   "Run pgxman in a container",
		Aliases: []string{"c"},
	}

	root.AddCommand(newContainerInstallOrUpgradeCmd("pgxman container", false))
	root.AddCommand(newContainerInstallOrUpgradeCmd("pgxman container", true))
	root.AddCommand(newContainerTeardownCmd())

	return root
}

func newContainerInstallOrUpgradeCmd(cmdPrefix string, upgrade bool) *cobra.Command {
	action := "install"
	if upgrade {
		action = "upgrade"
	}

	exampleTmpl := `  # {{ title .Action }} the latest pgvector in a container.
  {{ .Command }} {{ .Action }} pgvector

  # {{ title .Action }} the latest pgvector for PostgreSQL {{ .PGVer }} in a container.
  {{ .Command }} {{ .Action }} pgvector --pg {{ .PGVer }}

  # {{ title .Action }} pgvector 0.5.0 for PostgreSQL {{ .PGVer }} in a container.
  {{ .Command }} {{ .Action }} pgvector=0.5.0 --pg {{ .PGVer }}

  # {{ title .Action }} pgvector 0.5.0 and postgis 3.3.3 for PostgreSQL {{ .PGVer }} in a container
  {{ .Command }} {{ .Action }} pgvector=0.5.0 postgis=3.3.3 --pg {{ .PGVer }}

  # {{ title .Action }} a local Debian package in a container
  {{ .Command }} {{ .Action }} /PATH_TO/postgresql-15-pgxman-pgvector_0.5.0_arm64.deb`

	type data struct {
		Command string
		Action  string
		PGVer   string
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
			Command: cmdPrefix,
			Action:  action,
			PGVer:   string(pgxman.DefaultPGVersion),
		},
	); err != nil {
		// impossible
		panic(err.Error())
	}

	cmd := &cobra.Command{
		Use:   action,
		Short: c.String(action) + " PostgreSQL extensions in a container",
		Long: fmt.Sprintf(`Start a container with the specified PostgreSQL version and %s
PostgreSQL extension from commandline arguments. The argument format
is NAME=VERSION.`, action),
		Example: buf.String(),
		RunE:    runContainerInstall(upgrade),
		Args:    cobra.MinimumNArgs(1),
	}

	defPGVer := string(pgxman.DefaultPGVersion)

	cmd.PersistentFlags().StringVar(&flagContainerInstallPGVersion, "pg", defPGVer, fmt.Sprintf(c.String(action)+" the extension for the PostgreSQL version. Supported values are %s.", strings.Join(supportedPGVersions(), ", ")))
	cmd.PersistentFlags().StringVar(&flagContainerInstallRunnerImage, "runner-image", "", "Override the default runner image")

	return cmd
}

func runContainerInstall(upgrade bool) func(c *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		p := &ArgsParser{
			PGVer:  pgxman.PGVersion(flagContainerInstallPGVersion),
			Logger: log.NewTextLogger(),
		}
		f, err := p.Parse(cmd.Context(), args)
		if err != nil {
			return err
		}

		var (
			action = "Installing"
		)
		if upgrade {
			action = "Upgrading"
		}

		var (
			c = container.NewContainer(
				container.WithRunnerImage(flagContainerInstallRunnerImage),
				container.WithConfigDir(config.ConfigDir()),
				container.WithDebug(flagDebug),
			)
			info *container.ContainerInfo
		)

		fmt.Printf("%s extensions in a container for PostgreSQL %s...\n", action, flagContainerInstallPGVersion)
		for _, ext := range f {
			var err error
			info, err = installInContainer(cmd.Context(), c, ext)
			if err != nil {
				return err
			}
		}

		fmt.Printf(`To connect, run:

    $ psql postgres://%s:%s@127.0.0.1:%s/%s

To tear down the container, run:

    $ pgxman container teardown %s

For more information on the docker environment, please see: https://docs.pgxman.com/container.
`,
			info.Postgres.Username,
			info.Postgres.Password,
			info.Postgres.Port,
			info.Postgres.DBName,
			info.ContainerName,
		)

		return nil
	}
}

func newContainerTeardownCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teardown",
		Short: "Tear down a container",
		Long:  `Tear down a container and purge all data.`,
		Example: ` # Tear down the PostgreSQL 15 playgrond container.
pgxman container teardown pgxman_runner_15

# Tear down the PostgreSQL 15 & 16 containers.
pgxman container teardown pgxman_runner_15 pgxman_runner_16
`,
		RunE: runContainerTeardown,
		Args: cobra.MinimumNArgs(1),
	}

	return cmd
}

var (
	regexpContainerName = regexp.MustCompile(`^pgxman_runner_(\d+)$`)
)

func runContainerTeardown(cmd *cobra.Command, args []string) error {
	c := container.NewContainer(
		container.WithConfigDir(config.ConfigDir()),
		container.WithDebug(flagDebug),
	)
	for _, arg := range args {
		match := regexpContainerName.FindStringSubmatch(arg)
		if len(match) == 0 {
			return fmt.Errorf("invalid container name: %s", arg)
		}

		pgVer := pgxman.PGVersion(match[1])
		if err := pgxman.ValidatePGVersion(pgVer); err != nil {
			return err
		}

		s := newSpinner()
		s.Suffix = fmt.Sprintf(" Tearing down container for PostgreSQL %s...\n", pgVer)
		s.Start()
		if err := c.Teardown(cmd.Context(), pgVer); err != nil {
			return err
		}
		s.Stop()
	}

	return nil
}

func installInContainer(ctx context.Context, c *container.Container, ext pgxman.InstallExtension) (*container.ContainerInfo, error) {
	s := newSpinner()
	s.Suffix = fmt.Sprintf(" Installing %s...\n", ext)
	defer s.Stop()

	s.Start()
	info, err := c.Install(ctx, ext)
	if err != nil {
		s.FinalMSG = fmt.Sprintf("[%s] %s\n", errorMark, ext)
		return nil, fmt.Errorf("failed to install %s in a container, run with `--debug` to see the full error: %w", ext, err)
	}

	s.FinalMSG = fmt.Sprintf("[%s] %s: https://pgx.sh/%s\n", successMark, ext, ext.Name)

	return info, nil
}

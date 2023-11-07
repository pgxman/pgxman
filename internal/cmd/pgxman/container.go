package pgxman

import (
	"fmt"
	"strings"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/container"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/spf13/cobra"
)

func newContainerCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "container",
		Short:   "Run virtualized playground in a container",
		Aliases: []string{"c"},
	}

	root.AddCommand(newContainerInstallCmd())

	return root
}

func newContainerInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install PostgreSQL extensions in a container",
		Long: `Start a container with the specified PostgreSQL version and install
specified PostgreSQL extension from commandline arguments.`,
		Example: ` # Install the latest pgvector for the installed PostgreSQL.
		`,
		RunE: runContainerInstall,
	}

	cmd.PersistentFlags().StringVar(&flagInstallerPGVersion, "pg", string(pgxman.SupportedLatestPGVersion), "Install the extension for the PostgreSQL version.")

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
		if ext.Path != "" {
			return fmt.Errorf("cannot install extension %s from path in container", ext.Name)
		}

		exts = append(exts, ext.Name)
	}

	info, err := container.NewContainer(
		container.ContainerConfig{},
	).Install(cmd.Context(), f)
	if err != nil {
		return err
	}

	fmt.Printf(`%s installed successfully.

pgxman is running in a Docker container. To connect, run:

    $ psql postgres://%s:%s@127.0.0.1:%s/%s

To stop the container, run:

    $ cd "%s" && docker compose down && cd -

For more information on the docker environment, please see: https://docs.pgxman.com/container
`,
		strings.Join(exts, ", "),
		info.PGUser,
		info.PGPassword,
		info.Port,
		info.PGDatabase,
		info.RunnerDir,
	)

	return nil
}

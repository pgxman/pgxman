package pgxm

import "github.com/spf13/cobra"

func newPackageCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "package",
		Aliases: []string{"pkg", "p"},
		Short:   "Package extension according to the configuration file",
		RunE:    runPackage,
	}

	return cmd
}

func runPackage(cmd *cobra.Command, args []string) error {
	return nil
}

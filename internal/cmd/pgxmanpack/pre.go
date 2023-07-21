package pgxmanpack

import (
	"github.com/spf13/cobra"
)

func newPreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pre",
		Short: "Run pre-build tasks",
		RunE:  runPre,
	}

	return cmd
}

func runPre(cmd *cobra.Command, args []string) error {
	if err := packager.Pre(
		cmd.Context(),
		extension,
		packagerOpts,
	); err != nil {
		return err
	}

	return nil
}

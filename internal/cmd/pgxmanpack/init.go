package pgxmanpack

import (
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Run init build tasks",
		RunE:  runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	if err := packager.Init(
		cmd.Context(),
		extension,
		packagerOpts,
	); err != nil {
		return err
	}

	return nil
}

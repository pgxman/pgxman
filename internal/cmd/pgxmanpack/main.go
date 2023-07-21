package pgxmanpack

import (
	"github.com/spf13/cobra"
)

func newMainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "main",
		Short: "Run main build tasks",
		RunE:  runMain,
	}

	return cmd
}

func runMain(cmd *cobra.Command, args []string) error {
	if err := packager.Main(
		cmd.Context(),
		extension,
		packagerOpts,
	); err != nil {
		return err
	}

	return nil
}

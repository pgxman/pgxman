package pgxmanpack

import (
	"github.com/spf13/cobra"
)

func newPostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post",
		Short: "Run post-build tasks",
		RunE:  runPost,
	}

	return cmd
}

func runPost(cmd *cobra.Command, args []string) error {
	if err := packager.Post(
		cmd.Context(),
		extension,
		packagerOpts,
	); err != nil {
		return err
	}

	return nil
}

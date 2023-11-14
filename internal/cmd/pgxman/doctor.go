package pgxman

import "github.com/spf13/cobra"

func newDoctorCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "doctor",
		Short: "Troubleshoot your system for potential problems",
		Long:  `Check your system for potential problems. Will exit with a non-zero status if any potential problems are found.`,
		RunE:  runDoctor,
	}

	return root
}

func runDoctor(cmd *cobra.Command, args []string) error {
	return nil
}

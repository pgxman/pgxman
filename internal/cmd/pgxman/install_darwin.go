//go:build darwin

package pgxman

import "github.com/spf13/cobra"

func newInstallCmd() *cobra.Command {
	return newContainerInstallCmd()
}

func newUpgradeCmd() *cobra.Command {
	return newContainerInstallCmd()
}

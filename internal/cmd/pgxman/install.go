//go:build !linux

package pgxman

import "github.com/spf13/cobra"

func newInstallCmd() *cobra.Command {
	return newContainerInstallOrUpgradeCmd("pgxman", false)
}

func newUpgradeCmd() *cobra.Command {
	return newContainerInstallOrUpgradeCmd("pgxman", true)
}

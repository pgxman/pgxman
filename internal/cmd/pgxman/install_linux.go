//go:build linux

package pgxman

import "github.com/spf13/cobra"

func newInstallCmd() *cobra.Command {
	return newInstallOrUpgradeCmd(false)
}

func newUpgradeCmd() *cobra.Command {
	return newInstallOrUpgradeCmd(true)
}

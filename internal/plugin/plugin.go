package plugin

import (
	"fmt"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"

	"github.com/pgxman/pgxman/internal/plugin/darwin"
	"github.com/pgxman/pgxman/internal/plugin/debian"
)

func init() {
	debPkg := &debian.DebianPackager{
		Logger: log.NewTextLogger(),
	}
	RegisterPackager(pgxman.ExtensionBuilderDebianBookworm, debPkg)
	RegisterPackager(pgxman.ExtensionBuilderUbuntuJammy, debPkg)

	debUpdater := &debian.DebianUpdater{
		Logger: log.NewTextLogger(),
	}
	RegisterUpdater(pgxman.ExtensionBuilderDebianBookworm, debUpdater)
	RegisterUpdater(pgxman.ExtensionBuilderUbuntuJammy, debUpdater)

	debInstaller := &debian.DebianInstaller{
		Logger: log.NewTextLogger(),
	}
	RegisterInstaller(pgxman.ExtensionBuilderDebianBookworm, debInstaller)
	RegisterInstaller(pgxman.ExtensionBuilderUbuntuJammy, debInstaller)

	darwinUpdater := &darwin.DarwinUpdater{
		Logger: log.NewTextLogger(),
	}
	RegisterUpdater(pgxman.ExtensionBuilderDarwin, darwinUpdater)
}

var (
	packagers  = make(map[pgxman.ExtensionBuilderType]pgxman.Packager)
	updaters   = make(map[pgxman.ExtensionBuilderType]pgxman.Updater)
	installers = make(map[pgxman.ExtensionBuilderType]pgxman.Installer)
)

func RegisterPackager(bt pgxman.ExtensionBuilderType, packager pgxman.Packager) {
	packagers[bt] = packager
}

func GetPackager() (pgxman.Packager, error) {
	bt, err := pgxman.DetectExtensionBuilder()
	if err != nil {
		return nil, err
	}

	pkg := packagers[bt]
	if pkg == nil {
		return nil, &ErrUnsupportedPlugin{bt: bt}
	}

	return pkg, nil
}

func RegisterUpdater(bt pgxman.ExtensionBuilderType, updater pgxman.Updater) {
	updaters[bt] = updater
}

func GetUpdater() (pgxman.Updater, error) {
	bt, err := pgxman.DetectExtensionBuilder()
	if err != nil {
		return nil, err
	}

	u := updaters[bt]
	if u == nil {
		return nil, &ErrUnsupportedPlugin{bt: bt}
	}

	return u, nil
}

func RegisterInstaller(bt pgxman.ExtensionBuilderType, installer pgxman.Installer) {
	installers[bt] = installer
}

func GetInstaller() (pgxman.Installer, error) {
	bt, err := pgxman.DetectExtensionBuilder()
	if err != nil {
		return nil, err
	}

	i := installers[bt]
	if i == nil {
		return nil, &ErrUnsupportedPlugin{bt: bt}
	}

	return i, nil
}

type ErrUnsupportedPlugin struct {
	bt pgxman.ExtensionBuilderType
}

func (e *ErrUnsupportedPlugin) Error() string {
	return fmt.Sprintf("Unsupported plugin: %s", e.bt)
}

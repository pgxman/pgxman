package plugin

import (
	"fmt"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"

	"github.com/pgxman/pgxman/internal/plugin/debian"
)

func init() {
	debPkg := &debian.DebianPackager{
		Logger: log.NewTextLogger(),
	}
	RegisterPackager(pgxman.PlatformDebianBookworm, debPkg)
	RegisterPackager(pgxman.PlatformUbuntuJammy, debPkg)
	RegisterPackager(pgxman.PlatformUbuntuNoble, debPkg)

	debInstaller := &debian.DebianInstaller{
		Logger: log.NewTextLogger(),
	}
	RegisterInstaller(pgxman.PlatformDebianBookworm, debInstaller)
	RegisterInstaller(pgxman.PlatformUbuntuJammy, debInstaller)
	RegisterInstaller(pgxman.PlatformUbuntuNoble, debInstaller)
}

var (
	packagers  = make(map[pgxman.Platform]pgxman.Packager)
	installers = make(map[pgxman.Platform]pgxman.Installer)
)

func RegisterPackager(p pgxman.Platform, packager pgxman.Packager) {
	packagers[p] = packager
}

func GetPackager() (pgxman.Packager, error) {
	bt, err := pgxman.DetectPlatform()
	if err != nil {
		return nil, err
	}

	pkg := packagers[bt]
	if pkg == nil {
		return nil, &ErrUnsupportedPlugin{p: bt}
	}

	return pkg, nil
}

func RegisterInstaller(p pgxman.Platform, installer pgxman.Installer) {
	installers[p] = installer
}

func GetInstaller() (pgxman.Installer, error) {
	bt, err := pgxman.DetectPlatform()
	if err != nil {
		return nil, err
	}

	i := installers[bt]
	if i == nil {
		return nil, &ErrUnsupportedPlugin{p: bt}
	}

	return i, nil
}

type ErrUnsupportedPlugin struct {
	p pgxman.Platform
}

func (e *ErrUnsupportedPlugin) Error() string {
	return fmt.Sprintf("Unsupported plugin: %s", e.p)
}

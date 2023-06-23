package plugin

import (
	"fmt"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/osx"

	"github.com/pgxman/pgxman/internal/plugin/debian"
)

func init() {
	pkg := &debian.DebianPackager{
		Logger: log.NewTextLogger(),
	}
	RegisterPackager("debian", pkg)
	RegisterPackager("ubuntu", pkg)

	updater := &debian.DebianUpdater{
		Logger: log.NewTextLogger(),
	}
	RegisterUpdater("debian", updater)
	RegisterUpdater("ubuntu", updater)
}

var (
	packagers = make(map[string]pgxman.Packager)
	updaters  = make(map[string]pgxman.Updater)
)

func RegisterPackager(os string, packager pgxman.Packager) {
	packagers[os] = packager
}

func GetPackager() (pgxman.Packager, error) {
	os := osx.Vendor()
	pkg := packagers[os]
	if pkg == nil {
		return nil, ErrUnsupportedOS{os: os}
	}

	return pkg, nil
}

func RegisterUpdater(os string, updater pgxman.Updater) {
	updaters[os] = updater
}

func GetUpdater() (pgxman.Updater, error) {
	os := osx.Vendor()
	u := updaters[os]
	if u == nil {
		return nil, ErrUnsupportedOS{os: os}
	}

	return u, nil
}

type ErrUnsupportedOS struct {
	os string
}

func (e ErrUnsupportedOS) Error() string {
	return fmt.Sprintf("unsupported OS: %s", e.os)
}

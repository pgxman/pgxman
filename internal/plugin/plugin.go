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
	RegisterPackager(OSDebianBookworm, pkg)
	RegisterPackager(OSUbuntuJammy, pkg)

	updater := &debian.DebianUpdater{
		Logger: log.NewTextLogger(),
	}
	RegisterUpdater(OSDebianBookworm, updater)
	RegisterUpdater(OSUbuntuJammy, updater)

	installer := &debian.DebianInstaller{
		Logger: log.NewTextLogger(),
	}
	RegisterInstaller(OSDebianBookworm, installer)
	RegisterInstaller(OSUbuntuJammy, installer)
}

var (
	packagers  = make(map[OS]pgxman.Packager)
	updaters   = make(map[OS]pgxman.Updater)
	installers = make(map[OS]pgxman.Installer)
)

func RegisterPackager(os OS, packager pgxman.Packager) {
	packagers[os] = packager
}

func GetPackager() (pgxman.Packager, error) {
	os, err := detectOS()
	if err != nil {
		return nil, err
	}

	pkg := packagers[os]
	if pkg == nil {
		return nil, &ErrUnsupportedPlugin{os: os}
	}

	return pkg, nil
}

func RegisterUpdater(os OS, updater pgxman.Updater) {
	updaters[os] = updater
}

func GetUpdater() (pgxman.Updater, error) {
	os, err := detectOS()
	if err != nil {
		return nil, err
	}

	u := updaters[os]
	if u == nil {
		return nil, &ErrUnsupportedPlugin{os: os}
	}

	return u, nil
}

func RegisterInstaller(os OS, installer pgxman.Installer) {
	installers[os] = installer
}

func GetInstaller() (pgxman.Installer, error) {
	os, err := detectOS()
	if err != nil {
		return nil, err
	}

	i := installers[os]
	if i == nil {
		return nil, &ErrUnsupportedPlugin{os: os}
	}

	return i, nil
}

type ErrUnsupportedPlugin struct {
	os OS
}

func (e *ErrUnsupportedPlugin) Error() string {
	return fmt.Sprintf("Unsupported plugin: %s", e.os)
}

type OS string

const (
	OSUnsupported    OS = "unsupported"
	OSDebianBookworm OS = "debian:bookworm"
	OSUbuntuJammy    OS = "ubuntu:jammy"
)

type ErrUnsupportedOS struct {
	osVendor  string
	osVersion string
}

func (e *ErrUnsupportedOS) Error() string {
	return fmt.Sprintf("Unsupported OS: %s %s", e.osVendor, e.osVersion)
}

func detectOS() (OS, error) {
	info := osx.Sysinfo()

	var (
		vendor  = info.OS.Vendor
		version = info.OS.Version
	)

	if vendor == "debian" && version == "12" {
		return OSDebianBookworm, nil
	}

	if vendor == "ubuntu" && version == "22.04" {
		return OSUbuntuJammy, nil
	}

	return OSUnsupported, &ErrUnsupportedOS{osVendor: vendor, osVersion: version}
}

package pgxman

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
)

type InstallExtension struct {
	Name      string
	Version   string
	PGVersion PGVersion
}

func (e InstallExtension) Validate() error {
	if e.Name == "" {
		return fmt.Errorf("name is required")
	}

	if e.Version == "" {
		return fmt.Errorf("version is required")
	}

	if !slices.Contains(SupportedPGVersions, e.PGVersion) {
		return fmt.Errorf("unsupported pg version: %s", e.PGVersion)
	}

	return nil
}

type Installer interface {
	Install(ctx context.Context, ext []InstallExtension) error
}

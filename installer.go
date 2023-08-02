package pgxman

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
)

const DefaultInstallExtensionsAPIVersion = "v1"

type InstallExtensions struct {
	APIVersion string             `json:"apiVersion"`
	Extensions []InstallExtension `json:"extensions"`
	PGVersions []PGVersion        `json:"pgVersions"`
}

func (exts InstallExtensions) Validate() error {
	if exts.APIVersion != DefaultInstallExtensionsAPIVersion {
		return fmt.Errorf("invalid api version: %s", exts.APIVersion)
	}

	if len(exts.Extensions) > 0 && len(exts.PGVersions) == 0 {
		return fmt.Errorf("pgVersions is required")
	}

	for _, ext := range exts.Extensions {
		if err := ext.Validate(); err != nil {
			return err
		}
	}

	for _, pgv := range exts.PGVersions {
		if !slices.Contains(SupportedPGVersions, pgv) {
			return fmt.Errorf("unsupported pg version: %s", pgv)
		}
	}

	return nil
}

type InstallExtension struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Path    string `json:"path,omitempty"`
}

func (e InstallExtension) Validate() error {
	if e.Name == "" && e.Path == "" {
		return fmt.Errorf("name or path is required")
	}

	if e.Name != "" {
		if e.Version == "" {
			return fmt.Errorf("version is required")
		}
	}

	return nil
}

type Installer interface {
	Install(ctx context.Context, exts InstallExtensions) error
}

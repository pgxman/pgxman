package pgxman

import (
	"context"
	"fmt"

	"golang.org/x/exp/slices"
)

const DefaultPGXManfileAPIVersion = "v1"

type PGXManfile struct {
	APIVersion string             `json:"apiVersion"`
	Extensions []InstallExtension `json:"extensions"`
	PGVersions []PGVersion        `json:"pgVersions"`
}

func (exts PGXManfile) Validate() error {
	if exts.APIVersion != DefaultPGXManfileAPIVersion {
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
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Path    string   `json:"path,omitempty"`
	Options []string `json:"options,omitempty"`
}

func (e InstallExtension) Validate() error {
	if e.Name == "" && e.Path == "" {
		return fmt.Errorf("name or path is required")
	}

	return nil
}

func NewInstallerOptions(optFuncs []InstallerOptionsFunc) *InstallerOptions {
	opts := &InstallerOptions{}
	for _, f := range optFuncs {
		f(opts)
	}

	return opts
}

type InstallerOptions struct {
	IgnorePrompt bool
	Sudo         bool
}

type InstallerOptionsFunc func(*InstallerOptions)

func InstallOptWithIgnorePrompt(ignore bool) InstallerOptionsFunc {
	return func(ops *InstallerOptions) {
		ops.IgnorePrompt = ignore
	}
}

func InstallOptWithSudo(sudo bool) InstallerOptionsFunc {
	return func(ops *InstallerOptions) {
		ops.Sudo = sudo
	}
}

type Installer interface {
	Install(ctx context.Context, exts []PGXManfile, opts ...InstallerOptionsFunc) error
}

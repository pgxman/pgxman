package pgxman

import (
	"context"
	"fmt"
)

const DefaultBundleAPIVersion = "v1"

type Bundle struct {
	APIVersion string            `json:"apiVersion"`
	Extensions []BundleExtension `json:"extensions"`
	Postgres   Postgres          `json:"postgres"`
}

func (file Bundle) Validate() error {
	if file.APIVersion != DefaultBundleAPIVersion {
		return fmt.Errorf("invalid api version: %s", file.APIVersion)
	}

	for _, ext := range file.Extensions {
		if err := ext.Validate(); err != nil {
			return err
		}
	}

	if err := file.Postgres.Validate(); err != nil {
		return err
	}

	return nil
}

type BundleExtension struct {
	Name    string   `json:"name,omitempty"`
	Version string   `json:"version,omitempty"`
	Path    string   `json:"path,omitempty"`
	Options []string `json:"options,omitempty"`
}

func (e BundleExtension) Validate() error {
	if e.Name == "" && e.Path == "" {
		return fmt.Errorf("name or path is required")
	}

	return nil
}

type Postgres struct {
	Version  PGVersion `json:"version"`
	Username string    `json:"username,omitempty"`
	Password string    `json:"password,omitempty"`
	DBName   string    `json:"dbname,omitempty"`
	Port     string    `json:"port,omitempty"`
}

func (p Postgres) Validate() error {
	return ValidatePGVersion(p.Version)
}

func NewInstallerOptions(optFuncs []InstallerOptionsFunc) *InstallerOptions {
	opts := &InstallerOptions{}
	for _, f := range optFuncs {
		f(opts)
	}

	return opts
}

type InstallerOptions struct {
	BeforeRunHook func(debPkgs []string, sources []string) error
}

type InstallerOptionsFunc func(*InstallerOptions)

func WithBeforeRunHook(hook func(debPkgs []string, sources []string) error) InstallerOptionsFunc {
	return func(ops *InstallerOptions) {
		ops.BeforeRunHook = hook
	}
}

type Installer interface {
	Install(ctx context.Context, b Bundle, opts ...InstallerOptionsFunc) error
	Upgrade(ctx context.Context, b Bundle, opts ...InstallerOptionsFunc) error
}

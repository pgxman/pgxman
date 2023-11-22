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

func (b Bundle) Validate() error {
	if b.APIVersion != DefaultBundleAPIVersion {
		return fmt.Errorf("invalid api version: %s", b.APIVersion)
	}

	for _, ext := range b.Extensions {
		if err := ext.Validate(); err != nil {
			return err
		}
	}

	if err := b.Postgres.Validate(); err != nil {
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
	BeforeRunHook func() error
	IO            IO
	IgnorePrompt  bool
}

type InstallerOptionsFunc func(*InstallerOptions)

func WithBeforeRunHook(hook func() error) InstallerOptionsFunc {
	return func(ops *InstallerOptions) {
		ops.BeforeRunHook = hook
	}
}

func WithIO(io IO) InstallerOptionsFunc {
	return func(ops *InstallerOptions) {
		ops.IO = io
	}
}

func WithIgnorePrompt(ignore bool) InstallerOptionsFunc {
	return func(ops *InstallerOptions) {
		ops.IgnorePrompt = ignore
	}
}

type Installer interface {
	Install(ctx context.Context, b Bundle, opts ...InstallerOptionsFunc) error
	Upgrade(ctx context.Context, b Bundle, opts ...InstallerOptionsFunc) error
}

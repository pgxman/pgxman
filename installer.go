package pgxman

import (
	"context"
	"fmt"
)

const DefaultPGXManfileAPIVersion = "v1"

type PGXManfile struct {
	APIVersion string             `json:"apiVersion"`
	Extensions []InstallExtension `json:"extensions"`
	Postgres   Postgres           `json:"postgres"`
}

func (file PGXManfile) Validate() error {
	if file.APIVersion != DefaultPGXManfileAPIVersion {
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
	Install(ctx context.Context, ext PGXManfile, opts ...InstallerOptionsFunc) error
	Upgrade(ctx context.Context, exts PGXManfile, opts ...InstallerOptionsFunc) error
}

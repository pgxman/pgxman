package pgxman

import (
	"context"
	"fmt"
)

const DefaultPackAPIVersion = "v1"

type Pack struct {
	APIVersion string          `json:"apiVersion"`
	Extensions []PackExtension `json:"extensions"`
	Postgres   Postgres        `json:"postgres"`
}

func (p Pack) Validate() error {
	if p.APIVersion != DefaultPackAPIVersion {
		return fmt.Errorf("invalid api version: %s", p.APIVersion)
	}

	for _, ext := range p.Extensions {
		if err := ext.Validate(); err != nil {
			return err
		}
	}

	if err := p.Postgres.Validate(); err != nil {
		return err
	}

	return nil
}

func (p Pack) InstallExtensions() []InstallExtension {
	var exts []InstallExtension
	for _, ext := range p.Extensions {
		exts = append(exts, InstallExtension{
			PackExtension: ext,
			PGVersion:     p.Postgres.Version,
		})
	}

	return exts
}

type InstallExtension struct {
	PackExtension
	PGVersion PGVersion
}

func (e InstallExtension) String() string {
	if e.Name != "" {
		return fmt.Sprintf("%s %s", e.Name, e.Version)
	}

	return e.Path
}

func (e InstallExtension) Validate() error {
	if err := e.PackExtension.Validate(); err != nil {
		return err
	}

	if err := e.PGVersion.Validate(); err != nil {
		return err
	}

	return nil
}

type PackExtension struct {
	Name      string   `json:"name,omitempty"`
	Version   string   `json:"version,omitempty"`
	Path      string   `json:"path,omitempty"`
	Options   []string `json:"options,omitempty"`
	Overwrite bool     `json:"overwrite,omitempty"`
}

func (e PackExtension) Validate() error {
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
	return p.Version.Validate()
}

type Installer interface {
	Install(ctx context.Context, ext InstallExtension) error
	Upgrade(ctx context.Context, ext InstallExtension) error
	PreInstallCheck(ctx context.Context, exts []InstallExtension, io IO) error
	PreUpgradeCheck(ctx context.Context, exts []InstallExtension, io IO) error
}

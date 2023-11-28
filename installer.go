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

func (b Pack) Validate() error {
	if b.APIVersion != DefaultPackAPIVersion {
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

	if err := ValidatePGVersion(e.PGVersion); err != nil {
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
	return ValidatePGVersion(p.Version)
}

type Installer interface {
	Install(ctx context.Context, ext InstallExtension) error
	Upgrade(ctx context.Context, ext InstallExtension) error
	PreInstallCheck(ctx context.Context, exts []InstallExtension, io IO) error
	PreUpgradeCheck(ctx context.Context, exts []InstallExtension, io IO) error
}

package pgxman

import (
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/exp/slices"
)

const (
	defaultBuildImage = "ghcr.io/pgxman/builder"
)

func NewDefaultExtension() Extension {
	var buildImageVersion string
	if Version == "dev" {
		buildImageVersion = "latest"
	} else {
		buildImageVersion = fmt.Sprintf("v%s", Version)
	}

	return Extension{
		APIVersion: DefaultAPIVersion,
		PGVersions: SupportedPGVersions,
		Platform:   []Platform{PlatformLinux},
		Arch:       []Arch{Arch(runtime.GOARCH)},
		Formats:    []Format{FormatDeb},
		BuildImage: fmt.Sprintf("%s:%s", defaultBuildImage, buildImageVersion),
	}
}

type Extension struct {
	// required
	APIVersion  string       `json:"apiVersion"`
	Name        string       `json:"name"`
	Source      string       `json:"source"`
	Version     string       `json:"version"`
	PGVersions  []PGVersion  `json:"pgVersions"`
	Build       string       `json:"build"`
	Maintainers []Maintainer `json:"maintainers"`

	// optional
	Arch              []Arch     `json:"arch,omitempty"`
	Platform          []Platform `json:"platform,omitempty"`
	Formats           []Format   `json:"formats,omitempty"`
	Description       string     `json:"description,omitempty"`
	License           string     `json:"license,omitempty"`
	Keywords          []string   `json:"keywords,omitempty"`
	Homepage          string     `json:"homepage,omitempty"`
	BuildDependencies []string   `json:"buildDependencies,omitempty"`
	Dependencies      []string   `json:"dependencies,omitempty"`
	BuildImage        string     `json:"buildImage,omitempty"`

	// override
	Deb *Deb `json:"deb,omitempty"`
}

func (ext Extension) Validate() error {
	if ext.APIVersion != DefaultAPIVersion {
		return fmt.Errorf("invalid api version: %s", ext.APIVersion)
	}

	if ext.Name == "" {
		return fmt.Errorf("name is required")
	}

	if ext.Source == "" {
		return fmt.Errorf("source is required")
	}
	if !strings.HasSuffix(ext.Source, "tar.gz") {
		return fmt.Errorf("source only supports tar.gz format")
	}

	if ext.Version == "" {
		return fmt.Errorf("version is required")
	}

	if len(ext.PGVersions) == 0 {
		return fmt.Errorf("pgVersions is required")
	}

	if ext.Build == "" {
		return fmt.Errorf("build is required")
	}

	if len(ext.Maintainers) == 0 {
		return fmt.Errorf("maintainers is required")
	}

	if ext.BuildImage == "" {
		return fmt.Errorf("build image is required")
	}

	for _, pgv := range ext.PGVersions {
		if !slices.Contains(SupportedPGVersions, pgv) {
			return fmt.Errorf("unsupported pg version: %s", pgv)
		}
	}

	for _, a := range ext.Arch {
		if !slices.Contains(SupportedArchs, a) {
			return fmt.Errorf("unsupported arch: %s", a)
		}
	}

	for _, f := range ext.Formats {
		if !slices.Contains(SupportedFormats, f) {
			return fmt.Errorf("unsupported format: %s", f)
		}
	}

	for _, p := range ext.Platform {
		if !slices.Contains(SupprtedPlatforms, p) {
			return fmt.Errorf("unsupported platform: %s", p)
		}
	}

	return nil
}

const DefaultAPIVersion = "v1"

type Arch string

const (
	ArchAmd64 Arch = "amd64"
	ArchArm64 Arch = "arm64"
)

var (
	SupportedArchs = []Arch{ArchAmd64, ArchArm64}
)

type Format string

const (
	FormatDeb Format = "deb"
)

var (
	SupportedFormats = []Format{FormatDeb}
)

type Platform string

const (
	PlatformLinux Platform = "linux"
)

var (
	SupprtedPlatforms = []Platform{PlatformLinux}
)

type PGVersion string

const (
	PGVersion13 PGVersion = "13"
	PGVersion14 PGVersion = "14"
	PGVersion15 PGVersion = "15"
)

var (
	SupportedPGVersions = []PGVersion{PGVersion13, PGVersion14, PGVersion15}
)

type Maintainer struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Deb struct {
	BuildDependencies []string `json:"buildDependencies,omitempty"`
	Dependencies      []string `json:"dependencies,omitempty"`
}

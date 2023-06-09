package pgxm

import (
	"context"
	"crypto/sha1"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/hydradatabase/pgxm/internal/log"
	"github.com/imdario/mergo"
	"golang.org/x/exp/slices"
	"sigs.k8s.io/yaml"
)

func ReadExtensionFile(path string, overrides map[string]any) (Extension, error) {
	var ext Extension

	if _, err := os.Stat(path); err != nil {
		return ext, fmt.Errorf("extension.yaml not found in current directory")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return ext, err
	}

	b, err = overrideYamlFields(b, overrides)
	if err != nil {
		return ext, err
	}

	if err := yaml.Unmarshal(b, &ext); err != nil {
		return ext, err
	}

	ext = ext.WithDefaults()
	ext.ConfigSHA = fmt.Sprintf("%x", sha1.Sum(b))

	if err := ext.Validate(); err != nil {
		return ext, fmt.Errorf("invalid extension: %w", err)
	}

	return ext, nil
}

const (
	defaultBuildImage = "ghcr.io/hydradatabase/pgxm/builder"
)

type Extension struct {
	// required
	APIVersion  string       `json:"apiVersion"`
	Name        string       `json:"name"`
	Source      string       `json:"source"`
	Version     string       `json:"version"`
	PGVersions  []PGVersion  `json:"pgVersions"`
	Install     string       `json:"install"`
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
	Deb Deb `json:"deb,omitempty"`

	// internal
	ConfigSHA string `json:"configSHA,omitempty"`
}

func (ext *Extension) WithDefaults() Extension {
	if len(ext.PGVersions) == 0 {
		ext.PGVersions = SupportedPGVersions
	}

	if len(ext.Platform) == 0 {
		ext.Platform = []Platform{PlatformLinux}
	}

	if len(ext.Arch) == 0 {
		ext.Arch = []Arch{Arch(runtime.GOARCH)}
	}

	if ext.BuildImage == "" {
		ext.BuildImage = fmt.Sprintf("%s:%s", defaultBuildImage, Version)
	}

	return *ext
}

func (ext Extension) Validate() error {
	if ext.APIVersion != APIVersion {
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

	if ext.Install == "" {
		return fmt.Errorf("install is required")
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

const APIVersion = "v1"

type Arch string

const (
	ArchAmd64  Arch = "amd64"
	ArchAarm64 Arch = "arm64"
)

var (
	SupportedArchs = []Arch{ArchAmd64, ArchAarm64}
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

func NewPackager(workDir string) Packager {
	return &debianPackager{
		workDir: workDir,
		logger:  log.NewTextLogger(),
	}
}

type Packager interface {
	Package(ctx context.Context, ext Extension) error
}

func NewBuilder(extDir string) Builder {
	return &debianBuilder{
		extDir: extDir,
		logger: log.NewTextLogger(),
	}
}

type Builder interface {
	Build(ctx context.Context, ext Extension) error
}

func overrideYamlFields(b []byte, overrides map[string]any) ([]byte, error) {
	if len(overrides) == 0 {
		return b, nil
	}

	src := make(map[string]any)
	if err := yaml.Unmarshal(b, &src); err != nil {
		return nil, err
	}

	if err := mergo.Merge(&overrides, src); err != nil {
		return nil, err
	}

	return yaml.Marshal(overrides)
}

package pgxman

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/pgxman/pgxman/internal/osx"
	"golang.org/x/exp/slices"
)

func NewDefaultExtension() Extension {
	var buildImageVersion string
	if Version == "dev" {
		buildImageVersion = "latest"
	} else {
		buildImageVersion = fmt.Sprintf("v%s", Version)
	}

	return Extension{
		APIVersion: DefaultExtensionAPIVersion,
		PGVersions: SupportedPGVersions,
		Arch:       []Arch{Arch(runtime.GOARCH)},
		Platform:   []Platform{PlatformLinux},
		Formats:    []Format{FormatDeb},
		Builders: &ExtensionBuilders{
			DebianBookworm: &ExtensionBuilder{
				OS:    OSDebianBookworm,
				Image: fmt.Sprintf("%s:%s", extensionBuilderImages[OSDebianBookworm], buildImageVersion),
			},
			UbuntuJammy: &ExtensionBuilder{
				OS:    OSUbuntuJammy,
				Image: fmt.Sprintf("%s:%s", extensionBuilderImages[OSUbuntuJammy], buildImageVersion),
			},
		},
	}
}

type Extension struct {
	// required
	APIVersion  string       `json:"apiVersion"`
	Name        string       `json:"name"`
	Source      string       `json:"source"`
	Version     string       `json:"version"`
	PGVersions  []PGVersion  `json:"pgVersions"`
	Build       Build        `json:"build"`
	Maintainers []Maintainer `json:"maintainers"`

	// optional
	Builders          *ExtensionBuilders `json:"builders,omitempty"`
	Arch              []Arch             `json:"arch,omitempty"`
	Platform          []Platform         `json:"platform,omitempty"`
	Formats           []Format           `json:"formats,omitempty"`
	Description       string             `json:"description,omitempty"`
	License           string             `json:"license,omitempty"`
	Keywords          []string           `json:"keywords,omitempty"`
	Homepage          string             `json:"homepage,omitempty"`
	BuildDependencies []string           `json:"buildDependencies,omitempty"`
	Dependencies      []string           `json:"dependencies,omitempty"`
}

func (ext Extension) Validate() error {
	if ext.APIVersion != DefaultExtensionAPIVersion {
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

	_, err := semver.NewVersion(ext.Version)
	if err != nil {
		return fmt.Errorf("invalid semantic version: %w", err)
	}

	if len(ext.PGVersions) == 0 {
		return fmt.Errorf("pgVersions is required")
	}

	if err := ext.Build.Validate(); err != nil {
		return err
	}

	if len(ext.Maintainers) == 0 {
		return fmt.Errorf("maintainers is required")
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

	builders := ext.Builders.Items()
	if len(builders) == 0 {
		return fmt.Errorf("at least one extension builder is required")
	}

	for _, builder := range builders {
		if err := builder.Validate(); err != nil {
			return fmt.Errorf("builders.%s has errors: %w", builder.OS, err)
		}
	}

	return nil
}

const DefaultExtensionAPIVersion = "v1"

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

type Build struct {
	Pre  []BuildScript `json:"pre,omitempty"`
	Main []BuildScript `json:"main"`
	Post []BuildScript `json:"post,omitempty"`
}

func (b Build) Validate() error {
	if len(b.Main) == 0 {
		return fmt.Errorf("main build script is required")
	}

	for _, s := range b.Pre {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("pre-build script: %w", err)
		}
	}

	for _, s := range b.Main {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("main build script: %w", err)
		}
	}

	for _, s := range b.Post {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("post-build script: %w", err)
		}
	}

	return nil
}

type BuildScript struct {
	Name string `json:"name"`
	Run  string `json:"run"`
}

func (s BuildScript) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("build script name is required")
	}

	if s.Run == "" {
		return fmt.Errorf("build script run is required")
	}

	return nil
}

var (
	extensionBuilderImages = map[OS]string{
		OSDebianBookworm: "ghcr.io/pgxman/builder/debian/bookworm",
		OSUbuntuJammy:    "ghcr.io/pgxman/builder/ubuntu/jammy",
	}
)

type OS string

const (
	OSUnsupported    OS = "unsupported"
	OSDebianBookworm OS = "debian:bookworm"
	OSUbuntuJammy    OS = "ubuntu:jammy"
)

type ErrUnsupportedOS struct {
	osVendor  string
	osVersion string
}

func (e *ErrUnsupportedOS) Error() string {
	return fmt.Sprintf("Unsupported OS: %s %s", e.osVendor, e.osVersion)
}

type ExtensionBuilders struct {
	DebianBookworm *ExtensionBuilder `json:"debian:bookworm,omitempty"`
	UbuntuJammy    *ExtensionBuilder `json:"ubuntu:jammy,omitempty"`
}

func (ebs ExtensionBuilders) Items() []ExtensionBuilder {
	var result []ExtensionBuilder

	if builder := ebs.DebianBookworm; builder != nil {
		result = append(result, ebs.newBuilder(OSDebianBookworm, builder))
	}
	if builder := ebs.UbuntuJammy; builder != nil {
		result = append(result, ebs.newBuilder(OSUbuntuJammy, builder))
	}

	return result
}

func (ebs ExtensionBuilders) Current() ExtensionBuilder {
	os, err := DetectOS()

	if err != nil {
		panic(err.Error())
	}

	var builder *ExtensionBuilder
	switch os {
	case OSDebianBookworm:
		builder = ebs.DebianBookworm
	case OSUbuntuJammy:
		builder = ebs.UbuntuJammy
	}

	return ebs.newBuilder(os, builder)
}

func (ebs ExtensionBuilders) newBuilder(os OS, builder *ExtensionBuilder) ExtensionBuilder {
	image := builder.Image
	if image == "" {
		image = extensionBuilderImages[os]
	}

	return ExtensionBuilder{
		OS:                os,
		Image:             image,
		BuildDependencies: builder.BuildDependencies,
		RunDependencies:   builder.RunDependencies,
		AptRepositories:   builder.AptRepositories,
	}
}

type ExtensionBuilder struct {
	OS                OS              `json:"-"`
	Image             string          `json:"image,omitempty"`
	BuildDependencies []string        `json:"buildDependencies,omitempty"`
	RunDependencies   []string        `json:"runDependencies,omitempty"`
	AptRepositories   []AptRepository `json:"aptRepositories,omitempty"`
}

func (builder ExtensionBuilder) Validate() error {
	for i, repo := range builder.AptRepositories {
		if err := repo.Validate(); err != nil {
			return fmt.Errorf("aptRepositories[%d] has errors: %w", i, err)
		}
	}

	return nil
}

type Deb struct {
	BuildDependencies []string        `json:"buildDependencies,omitempty"`
	Dependencies      []string        `json:"dependencies,omitempty"`
	AptRepositories   []AptRepository `json:"aptRepositories,omitempty"`
}

// Ref: https://manpages.ubuntu.com/manpages/lunar/en/man5/sources.list.5.html
type AptRepository struct {
	ID         string                 `json:"id"`
	Types      []AptRepositoryType    `json:"types"`
	URIs       []string               `json:"uris"`
	Suites     []string               `json:"suites"`
	Components []string               `json:"components"`
	SignedKey  AptRepositorySignedKey `json:"signedKey"`
}

func (repo AptRepository) Validate() error {
	if repo.ID == "" {
		return fmt.Errorf("apt repository id is required")
	}

	if len(repo.Types) == 0 {
		return fmt.Errorf("apt repository types is required")
	}
	for _, t := range repo.Types {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("apt repository types: %w", err)
		}
	}

	if len(repo.URIs) == 0 {
		return fmt.Errorf("apt repository uris is required")
	}

	if len(repo.Suites) == 0 {
		return fmt.Errorf("apt repository suites is required")
	}

	if len(repo.Components) == 0 {
		return fmt.Errorf("apt repository components is required")
	}

	if err := repo.SignedKey.Validate(); err != nil {
		return fmt.Errorf("apt repository signed key: %w", err)
	}

	return nil
}

type AptRepositorySignedKey struct {
	URL    string                       `json:"url"`
	Format AptRepositorySignedKeyFormat `json:"format"`
}

func (k AptRepositorySignedKey) Validate() error {
	if k.URL == "" {
		return fmt.Errorf("url is required")
	}

	if !slices.Contains(SupportedAptRepositorySignedKeyFormats, k.Format) {
		return fmt.Errorf("unsupported format: %s", k.Format)
	}

	return nil
}

var (
	SupportedAptRepositoryTypes = []AptRepositoryType{AptRepositoryTypeDeb, AptRepositoryTypeDebSrc}
)

type AptRepositoryType string

func (t AptRepositoryType) Validate() error {
	if !slices.Contains(SupportedAptRepositoryTypes, t) {
		return fmt.Errorf("unsupported type: %s", t)
	}

	return nil
}

const (
	AptRepositoryTypeDeb    AptRepositoryType = "deb"
	AptRepositoryTypeDebSrc AptRepositoryType = "deb-src"
)

var (
	SupportedAptRepositorySignedKeyFormats = []AptRepositorySignedKeyFormat{AptRepositorySignedKeyFormatAsc, AptRepositorySignedKeyFormatGpg}
)

type AptRepositorySignedKeyFormat string

const (
	AptRepositorySignedKeyFormatAsc AptRepositorySignedKeyFormat = "asc"
	AptRepositorySignedKeyFormatGpg AptRepositorySignedKeyFormat = "gpg"
)

func DetectOS() (OS, error) {
	info := osx.Sysinfo()

	var (
		vendor  = info.OS.Vendor
		version = info.OS.Version
	)

	if vendor == "debian" && version == "12" {
		return OSDebianBookworm, nil
	}

	if vendor == "ubuntu" && version == "22.04" {
		return OSUbuntuJammy, nil
	}

	return OSUnsupported, &ErrUnsupportedOS{osVendor: vendor, osVersion: version}
}

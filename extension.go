package pgxman

import (
	"fmt"
	"net/url"
	"path/filepath"
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
			DebianBookworm: &AptExtensionBuilder{
				ExtensionBuilder: ExtensionBuilder{
					Type:  ExtensionBuilderDebianBookworm,
					Image: fmt.Sprintf("%s:%s", extensionBuilderImages[ExtensionBuilderDebianBookworm], buildImageVersion),
				},
			},
			UbuntuJammy: &AptExtensionBuilder{
				ExtensionBuilder: ExtensionBuilder{
					Type:  ExtensionBuilderUbuntuJammy,
					Image: fmt.Sprintf("%s:%s", extensionBuilderImages[ExtensionBuilderUbuntuJammy], buildImageVersion),
				},
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
	RunDependencies   []string           `json:"runDependencies,omitempty"`

	// internal
	Path string `json:"-"`
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
	_, err := ext.ParseSource()
	if err != nil {
		return fmt.Errorf("invalid source: %w", err)
	}

	if ext.Version == "" {
		return fmt.Errorf("version is required")
	}

	_, err = semver.NewVersion(ext.Version)
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

	builders := ext.Builders.Available()
	if len(builders) == 0 {
		return fmt.Errorf("at least one extension builder is required")
	}

	for _, builder := range builders {
		if err := builder.Validate(); err != nil {
			return fmt.Errorf("builders.%s has errors: %w", builder.Type, err)
		}
	}

	if ext.Path == "" {
		return fmt.Errorf("path is required")
	}

	return nil
}

func (ext Extension) ParseSource() (string, error) {
	u, err := url.ParseRequestURI(ext.Source)
	if err != nil {
		return "", err
	}

	supportedScheme := []string{"http", "https", "file", ""}
	if !slices.Contains(supportedScheme, u.Scheme) {
		return "", fmt.Errorf("source only supports %s", strings.Join(supportedScheme, ", "))
	}

	if u.Scheme == "http" || u.Scheme == "https" {
		if !strings.HasSuffix(u.Path, ".tar.gz") {
			return "", fmt.Errorf("http source only supports tar.gz format: %s", u.Path)
		}

		return u.String(), nil
	}

	var path string
	if filepath.IsAbs(u.Path) {
		path = u.Path
	} else {
		// relative path to the buildkit file
		path = filepath.Join(filepath.Dir(ext.Path), u.Path)
	}

	return filepath.Clean(path), nil
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
	extensionBuilderImages = map[ExtensionBuilderType]string{
		ExtensionBuilderDebianBookworm: "ghcr.io/pgxman/builder/debian/bookworm",
		ExtensionBuilderUbuntuJammy:    "ghcr.io/pgxman/builder/ubuntu/jammy",
	}
)

type ExtensionBuilderType string

const (
	ExtensionBuilderUnsupported    ExtensionBuilderType = "unsupported"
	ExtensionBuilderDebianBookworm ExtensionBuilderType = "debian:bookworm"
	ExtensionBuilderUbuntuJammy    ExtensionBuilderType = "ubuntu:jammy"
)

type ErrUnsupportedExtensionBuilder struct {
	osVendor  string
	osVersion string
}

func (e *ErrUnsupportedExtensionBuilder) Error() string {
	builder := e.osVendor
	if e.osVersion != "" {
		builder += ":" + e.osVersion
	}

	return fmt.Sprintf("Unsupported builder: %s", builder)
}

type ExtensionBuilders struct {
	DebianBookworm *AptExtensionBuilder `json:"debian:bookworm,omitempty"`
	UbuntuJammy    *AptExtensionBuilder `json:"ubuntu:jammy,omitempty"`
}

func (ebs ExtensionBuilders) HasBuilder(bt ExtensionBuilderType) bool {
	switch bt {
	case ExtensionBuilderDebianBookworm:
		return ebs.DebianBookworm != nil
	case ExtensionBuilderUbuntuJammy:
		return ebs.UbuntuJammy != nil
	}

	return false
}

// Available returns all available extension builders.
func (ebs ExtensionBuilders) Available() []AptExtensionBuilder {
	var result []AptExtensionBuilder

	if builder := ebs.DebianBookworm; builder != nil {
		result = append(result, ebs.newBuilder(ExtensionBuilderDebianBookworm, builder))
	}
	if builder := ebs.UbuntuJammy; builder != nil {
		result = append(result, ebs.newBuilder(ExtensionBuilderUbuntuJammy, builder))
	}

	return result
}

// Current returns the extension builder for the current os.
// It panics if no extension builder is available.
func (ebs ExtensionBuilders) Current() AptExtensionBuilder {
	bt, err := DetectExtensionBuilder()
	if err != nil {
		panic(err.Error())
	}

	var builder *AptExtensionBuilder
	switch bt {
	case ExtensionBuilderDebianBookworm:
		builder = ebs.DebianBookworm
	case ExtensionBuilderUbuntuJammy:
		builder = ebs.UbuntuJammy
	}

	return ebs.newBuilder(bt, builder)
}

func (ebs ExtensionBuilders) newBuilder(os ExtensionBuilderType, builder *AptExtensionBuilder) AptExtensionBuilder {
	image := builder.Image
	if image == "" {
		image = extensionBuilderImages[os]
	}

	return AptExtensionBuilder{
		ExtensionBuilder: ExtensionBuilder{
			Type:              os,
			Image:             image,
			BuildDependencies: builder.BuildDependencies,
			RunDependencies:   builder.RunDependencies,
		},
		AptRepositories: builder.AptRepositories,
	}
}

type ExtensionBuilder struct {
	Type              ExtensionBuilderType `json:"-"`
	Image             string               `json:"image,omitempty"`
	BuildDependencies []string             `json:"buildDependencies,omitempty"`
	RunDependencies   []string             `json:"runDependencies,omitempty"`
}

type AptExtensionBuilder struct {
	ExtensionBuilder

	AptRepositories []AptRepository `json:"aptRepositories,omitempty"`
}

func (builder AptExtensionBuilder) Validate() error {
	for i, repo := range builder.AptRepositories {
		if err := repo.Validate(); err != nil {
			return fmt.Errorf("aptRepositories[%d] has errors: %w", i, err)
		}
	}

	return nil
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

func DetectExtensionBuilder() (ExtensionBuilderType, error) {
	info := osx.Sysinfo()

	var (
		vendor  = info.OS.Vendor
		version = info.OS.Version
	)

	if vendor == "" {
		vendor = runtime.GOOS
	}

	if vendor == "debian" && version == "12" {
		return ExtensionBuilderDebianBookworm, nil
	}

	if vendor == "ubuntu" && version == "22.04" {
		return ExtensionBuilderUbuntuJammy, nil
	}

	return ExtensionBuilderUnsupported, &ErrUnsupportedExtensionBuilder{osVendor: vendor, osVersion: version}
}

package pgxman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/registry"
	"github.com/pgxman/pgxman/oapi"
)

type errInvalidExtensionFormat struct {
	Arg string
}

func (e errInvalidExtensionFormat) Error() string {
	return fmt.Sprintf("invalid extension format: %q. The format is NAME=VERSION...", e.Arg)
}

type ErrExtNotFound struct {
	Name string
}

func (e *ErrExtNotFound) Error() string {
	return fmt.Sprintf("extension %q not found", e.Name)
}

type ErrExtVerNotFound struct {
	Name    string
	Version string
}

func (e *ErrExtVerNotFound) Error() string {
	return fmt.Sprintf("extension %q version %q not found", e.Name, e.Version)
}

type ErrExtIncompatiblePG struct {
	Name      string
	PGVersion pgxman.PGVersion
}

func (e *ErrExtIncompatiblePG) Error() string {
	return fmt.Sprintf("extension %q is incompatible with PostgreSQL %s", e.Name, e.PGVersion)
}

type ErrExtIncompatiblePlatform struct {
	Name     string
	Version  string
	Platform pgxman.Platform
}

func (e *ErrExtIncompatiblePlatform) Error() string {
	return fmt.Sprintf("extension %q version %q is incompatible with platform %s", e.Name, e.Version, e.Platform)
}

// ContainerPlatformDetector returns a platform detector for the container environment which is alwasys Debian Bookworm.
func ContainerPlatformDetector() (pgxman.Platform, error) {
	return pgxman.PlatformDebianBookworm, nil
}

func DefaultPlatformDetector() (pgxman.Platform, error) {
	return pgxman.DetectPlatform()
}

func NewArgsParser(c registry.Client, d PlatformDetector, pgver pgxman.PGVersion, overwrite bool) *ArgsParser {
	return &ArgsParser{
		Client:           c,
		PlatformDetector: d,
		PGVer:            pgver,
		Overwrite:        overwrite,
		Logger:           log.NewTextLogger(),
	}
}

type PlatformDetector func() (pgxman.Platform, error)

type ArgsParser struct {
	Client           registry.Client
	PlatformDetector PlatformDetector
	Logger           *log.Logger
	PGVer            pgxman.PGVersion
	Overwrite        bool
}

func (p *ArgsParser) Parse(ctx context.Context, args []string) ([]pgxman.InstallExtension, error) {
	if err := p.PGVer.Validate(); err != nil {
		return nil, err
	}

	var exts []pgxman.InstallExtension
	for _, arg := range args {
		ext, err := parseInstallExtension(arg)
		if err != nil {
			return nil, err
		}
		ext.Overwrite = p.Overwrite

		exts = append(exts, pgxman.InstallExtension{
			PackExtension: *ext,
			PGVersion:     p.PGVer,
		})
	}

	locker := NewExtensionLocker(p.Client, p.PlatformDetector, p.Logger)
	return locker.Lock(ctx, exts)
}

func NewExtensionLocker(c registry.Client, d PlatformDetector, logger *log.Logger) *ExtensionLocker {
	return &ExtensionLocker{
		Client:           c,
		PlatformDetector: d,
		Logger:           logger,
	}
}

type ExtensionLocker struct {
	Client           registry.Client
	PlatformDetector PlatformDetector
	Logger           *log.Logger
}

func (l *ExtensionLocker) Lock(ctx context.Context, exts []pgxman.InstallExtension) ([]pgxman.InstallExtension, error) {
	p, err := l.PlatformDetector()
	if err != nil {
		return nil, fmt.Errorf("detect platform: %s", err)
	}

	var result []pgxman.InstallExtension
	for _, ext := range exts {
		if ext.Name != "" {
			installableExt, err := l.Client.GetExtension(ctx, ext.Name)
			if err != nil {
				if errors.Is(err, registry.ErrExtensionNotFound) {
					return nil, &ErrExtNotFound{Name: ext.Name}
				}
			}

			// if version is not specified, use the latest version
			if ext.Version != "" && ext.Version != "latest" {
				installableExt, err = l.Client.GetVersion(ctx, ext.Name, ext.Version)
				if err != nil {
					if errors.Is(err, registry.ErrExtensionNotFound) {
						return nil, &ErrExtVerNotFound{Name: ext.Name, Version: ext.Version}
					}

					return nil, err
				}
			}

			installablePkg, ok := installableExt.Packages[string(ext.PGVersion)]
			if !ok {
				return nil, &ErrExtIncompatiblePG{Name: ext.Name, PGVersion: ext.PGVersion}
			}
			ext.Version = installablePkg.Version

			platform, err := getPlatform(installablePkg, p)
			if err != nil {
				return nil, &ErrExtIncompatiblePlatform{Name: ext.Name, Version: ext.Version, Platform: p}
			}
			ext.AptRepositories = convertAptRepos(platform.AptRepositories)
		}

		result = append(result, ext)
	}

	return result, nil
}

func getPlatform(pkg oapi.Package, p pgxman.Platform) (*oapi.Platform, error) {
	for _, platform := range pkg.Platforms {
		if string(platform.Os) == string(p) {
			return &platform, nil
		}
	}

	return nil, fmt.Errorf("platform %q not found", p)
}

var (
	extRegexp = regexp.MustCompile(`^([^=@\s]+)(?:=([^@]*))?$`)
)

func parseInstallExtension(arg string) (*pgxman.PackExtension, error) {
	// install from local file
	if _, err := os.Stat(arg); err == nil {
		path, err := filepath.Abs(arg)
		if err != nil {
			return nil, err
		}

		return &pgxman.PackExtension{
			Path: path,
		}, nil
	}

	// install from apt
	if extRegexp.MatchString(arg) {
		var (
			match   = extRegexp.FindStringSubmatch(arg)
			name    = match[1]
			version = match[2]
		)

		return &pgxman.PackExtension{
			Name:    name,
			Version: version,
		}, nil
	}

	return nil, errInvalidExtensionFormat{Arg: arg}
}

func convertAptRepos(aptRepos []oapi.AptRepository) []pgxman.AptRepository {
	var result []pgxman.AptRepository
	for _, aptRepo := range aptRepos {
		result = append(result, convertAptRepo(aptRepo))
	}

	return result
}

func convertAptRepo(aptRepo oapi.AptRepository) pgxman.AptRepository {
	return pgxman.AptRepository{
		ID:         aptRepo.Id,
		Types:      convertAptRepoTypes(aptRepo.Types),
		URIs:       aptRepo.Uris,
		Components: aptRepo.Components,
		Suites:     aptRepo.Suites,
		SignedKey:  convertAptRepoSignedKey(aptRepo.SignedKey),
	}
}

func convertAptRepoTypes(types []oapi.AptRepositoryType) []pgxman.AptRepositoryType {
	var result []pgxman.AptRepositoryType
	for _, t := range types {
		var tt pgxman.AptRepositoryType
		switch t {
		case oapi.Deb:
			tt = pgxman.AptRepositoryTypeDeb
		case oapi.DebSrc:
			tt = pgxman.AptRepositoryTypeDebSrc
		default:
			panic(fmt.Sprintf("invalid apt repo type: %s", t))
		}
		result = append(result, tt)
	}

	return result
}

func convertAptRepoSignedKey(signedKey oapi.SignedKey) pgxman.AptRepositorySignedKey {
	var format pgxman.AptRepositorySignedKeyFormat
	switch signedKey.Format {
	case oapi.Gpg:
		format = pgxman.AptRepositorySignedKeyFormatGpg
	case oapi.Asc:
		format = pgxman.AptRepositorySignedKeyFormatAsc
	default:
		panic(fmt.Sprintf("invalid apt repo signed key format: %s", signedKey.Format))
	}

	return pgxman.AptRepositorySignedKey{
		URL:    signedKey.Url,
		Format: format,
	}
}

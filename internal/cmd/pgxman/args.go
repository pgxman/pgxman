package pgxman

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/registry"
	"github.com/pgxman/pgxman/oapi"
)

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
	Version   string
	PGVersion pgxman.PGVersion
}

func (e *ErrExtIncompatiblePG) Error() string {
	return fmt.Sprintf("extension %q version %q is incompatible with PostgreSQL %s", e.Name, e.Version, e.PGVersion)
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
	PlatformDetector PlatformDetector
	Client           registry.Client
	PGVer            pgxman.PGVersion
	Overwrite        bool
	Logger           *log.Logger
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
			if ext.Version == "" || ext.Version == "latest" {
				ext.Version = installableExt.Version
			}

			if installableExt.Version != ext.Version {
				installableExt, err = l.Client.GetVersion(ctx, ext.Name, ext.Version)
				if err != nil {
					if errors.Is(err, registry.ErrExtensionNotFound) {
						return nil, &ErrExtVerNotFound{Name: ext.Name, Version: ext.Version}
					}

					return nil, err
				}
			}

			platform, err := installableExt.GetPlatform(p)
			if err != nil {
				return nil, &ErrExtIncompatiblePlatform{Name: ext.Name, Version: ext.Version, Platform: p}
			}

			if !slices.Contains(platform.PgVersions, convertPGVersion(ext.PGVersion)) {
				return nil, &ErrExtIncompatiblePG{Name: ext.Name, Version: ext.Version, PGVersion: ext.PGVersion}
			}

			ext.AptRepositories = convertAptRepos(platform.AptRepositories)
		}

		result = append(result, ext)
	}

	return result, nil
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

func convertPGVersion(pgVer pgxman.PGVersion) oapi.PgVersion {
	switch pgVer {
	case pgxman.PGVersion13:
		return oapi.Pg13
	case pgxman.PGVersion14:
		return oapi.Pg14
	case pgxman.PGVersion15:
		return oapi.Pg15
	case pgxman.PGVersion16:
		return oapi.Pg16
	default:
		panic(fmt.Sprintf("invalid pg version: %s", pgVer))
	}
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

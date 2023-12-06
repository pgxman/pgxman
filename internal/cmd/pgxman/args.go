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

// ContainerPlatformDetector returns a platform detector for the container environment which is alwasys Debian Bookworm.
func ContainerPlatformDetector() (pgxman.Platform, error) {
	return pgxman.PlatformDebianBookworm, nil
}

func DefaultPlatformDetector() (pgxman.Platform, error) {
	return pgxman.DetectPlatform()
}

func NewArgsParser(d PlatformDetector, pgver pgxman.PGVersion, overwrite bool) *ArgsParser {
	return &ArgsParser{
		PlatformDetector: d,
		PGVer:            pgver,
		Overwrite:        overwrite,
		Logger:           log.NewTextLogger(),
	}
}

type PlatformDetector func() (pgxman.Platform, error)

type ArgsParser struct {
	PlatformDetector PlatformDetector
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

	locker, err := NewExtensionLocker(p.PlatformDetector, p.Logger)
	if err != nil {
		return nil, err
	}

	return locker.Lock(ctx, exts)
}

func NewExtensionLocker(d PlatformDetector, logger *log.Logger) (*ExtensionLocker, error) {
	c, err := registry.NewClient(flagRegistryURL)
	if err != nil {
		return nil, err
	}

	return &ExtensionLocker{
		Client:           c,
		PlatformDetector: d,
		Logger:           logger,
	}, nil
}

type ExtensionLocker struct {
	Client           *registry.Client
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
					return nil, fmt.Errorf("extension %q not found", ext.Name)
				}
			}

			// if version is not specified, use the latest version
			if ext.Version == "" || ext.Version == "latest" {
				ext.Version = installableExt.Version
			}

			if installableExt.Version != ext.Version {
				// TODO(owenthereal): validate old version when api is ready
				l.Logger.Debug("extension version does not match the latest", "extension", ext.Name, "version", ext.Version, "latest", installableExt.Version)
			}

			platform, err := installableExt.GetPlatform(p)
			if err != nil {
				return nil, err
			}

			if !slices.Contains(platform.PgVersions, convertPGVersion(ext.PGVersion)) {
				return nil, fmt.Errorf("%s %s is incompatible with PostgreSQL %s", ext.Name, ext.Version, ext.PGVersion)
			}
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

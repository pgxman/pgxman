package pgxman

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"github.com/pgxman/pgxman/internal/registry"
	"github.com/pgxman/pgxman/oapi"
	"github.com/stretchr/testify/assert"
)

func Test_parseInstallExtension(t *testing.T) {
	assert := assert.New(t)

	debFile := filepath.Join(t.TempDir(), "extension.deb")
	err := os.WriteFile(debFile, []byte{}, 0644)
	assert.NoError(err)

	cases := []struct {
		Name   string
		Arg    string
		GotExt *pgxman.PackExtension
		Err    error
	}{
		{
			Name: "valid with name & version",
			Arg:  "pgvector=0.5.0",
			GotExt: &pgxman.PackExtension{
				Name:    "pgvector",
				Version: "0.5.0",
			},
		},
		{
			Name: "valid with sha as version",
			Arg:  "parquet_s3_fdw=5298b7f0254923f52d15e554ec8a5fdc0474f059",
			GotExt: &pgxman.PackExtension{
				Name:    "parquet_s3_fdw",
				Version: "5298b7f0254923f52d15e554ec8a5fdc0474f059",
			},
		},
		{
			Name: "valid with empty version",
			Arg:  "pgvector=",
			GotExt: &pgxman.PackExtension{
				Name:    "pgvector",
				Version: "",
			},
		},
		{
			Name: "valid with latest as version",
			Arg:  "pgvector=latest",
			GotExt: &pgxman.PackExtension{
				Name:    "pgvector",
				Version: "latest",
			},
		},
		{
			Name: "valid with only name",
			Arg:  "pgvector",
			GotExt: &pgxman.PackExtension{
				Name:    "pgvector",
				Version: "",
			},
		},
		{
			Name: "valid file path",
			Arg:  debFile,
			GotExt: &pgxman.PackExtension{
				Path: debFile,
			},
		},
		{
			Name: "invalid",
			Arg:  "pgvector=0.5.0@",
			Err:  errInvalidExtensionFormat{Arg: "pgvector=0.5.0@"},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			gotExt, err := parseInstallExtension(c.Arg)
			assert.Equal(c.Err, err)
			assert.Equal(c.GotExt, gotExt)
		})
	}
}

func Test_ExtensionLocker(t *testing.T) {
	assert := assert.New(t)

	aptRepos := []oapi.AptRepository{
		{
			Id:         "pgdg",
			Types:      []oapi.AptRepositoryType{oapi.DebSrc, oapi.Deb},
			Uris:       []string{"https://apt.postgresql.org/pub/repos/apt"},
			Components: []string{"main"},
			Suites:     []string{"bookworm-pgdg"},
			SignedKey: oapi.SignedKey{
				Format: oapi.Asc,
				Url:    "https://www.postgresql.org/media/keys/ACCC4CF8.asc",
			},
		},
	}
	stubbedClient := StubbedRegistryClient{
		ExtGetExtension: &oapi.Extension{
			Name: "pgvector",
			Packages: oapi.Packages{
				string(pgxman.PGVersion16): {
					Version: "0.5.1",
					Platforms: []oapi.Platform{
						{
							Os:              oapi.DebianBookworm,
							AptRepositories: aptRepos,
						},
					},
				},
			},
		},
		ExtGetVersion: &oapi.Extension{
			Name: "pgvector",
			Packages: oapi.Packages{
				string(pgxman.PGVersion16): {
					Version: "0.5.0",
					Platforms: []oapi.Platform{
						{
							Os:              oapi.DebianBookworm,
							AptRepositories: aptRepos,
						},
					},
				},
			},
		},
	}
	stubbedPlatformDetector := func() (pgxman.Platform, error) {
		return pgxman.PlatformDebianBookworm, nil
	}

	cases := []struct {
		Name             string
		PlatformDetector PlatformDetector
		InstallExts      []pgxman.InstallExtension
		WantInstallExts  []pgxman.InstallExtension
		WantErr          error
	}{
		{
			Name:             "install old version",
			PlatformDetector: stubbedPlatformDetector,
			InstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name:    "pgvector",
						Version: "0.5.0",
					},
					PGVersion: pgxman.PGVersion16,
				},
			},
			WantInstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name:    "pgvector",
						Version: "0.5.0",
					},
					PGVersion:       pgxman.PGVersion16,
					AptRepositories: convertAptRepos(aptRepos),
				},
			},
		},
		{
			Name:             "install latest version",
			PlatformDetector: stubbedPlatformDetector,
			InstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name: "pgvector",
					},
					PGVersion: pgxman.PGVersion16,
				},
			},
			WantInstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name:    "pgvector",
						Version: "0.5.1",
					},
					PGVersion:       pgxman.PGVersion16,
					AptRepositories: convertAptRepos(aptRepos),
				},
			},
		},
		{
			Name:             "version doesn't exist",
			PlatformDetector: stubbedPlatformDetector,
			InstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name:    "pgvector",
						Version: "0.5.2",
					},
					PGVersion: pgxman.PGVersion16,
				},
			},
			WantErr: &ErrExtVerNotFound{Name: "pgvector", Version: "0.5.2"},
		},
		{
			Name:             "extension doesn't exist",
			PlatformDetector: stubbedPlatformDetector,
			InstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name:    "hello-world",
						Version: "0.5.2",
					},
					PGVersion: pgxman.PGVersion16,
				},
			},
			WantErr: &ErrExtNotFound{Name: "hello-world"},
		},
		{
			Name:             "extension incompatible with pg",
			PlatformDetector: stubbedPlatformDetector,
			InstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name: "pgvector",
					},
					PGVersion: pgxman.PGVersion13,
				},
			},
			WantErr: &ErrExtIncompatiblePG{Name: "pgvector", PGVersion: pgxman.PGVersion13},
		},
		{
			Name: "extension incompatible with platform",
			PlatformDetector: func() (pgxman.Platform, error) {
				return pgxman.PlatformDarwin, nil
			},
			InstallExts: []pgxman.InstallExtension{
				{
					PackExtension: pgxman.PackExtension{
						Name: "pgvector",
					},
					PGVersion: pgxman.PGVersion16,
				},
			},
			WantErr: &ErrExtIncompatiblePlatform{Name: "pgvector", Version: "0.5.1", Platform: pgxman.PlatformDarwin},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			locker := NewExtensionLocker(stubbedClient, c.PlatformDetector, log.NewTextLogger())
			gotExts, gotErr := locker.Lock(context.TODO(), c.InstallExts)

			assert.Equal(c.WantErr, gotErr)
			assert.Equal(c.WantInstallExts, gotExts)
		})
	}
}

type StubbedRegistryClient struct {
	ExtGetExtension *oapi.Extension
	ExtGetVersion   *oapi.Extension
}

func (s StubbedRegistryClient) GetExtension(ctx context.Context, name string) (*oapi.Extension, error) {
	if name == s.ExtGetExtension.Name {
		return s.ExtGetExtension, nil
	}

	return nil, registry.ErrExtensionNotFound
}

func (s StubbedRegistryClient) FindExtension(ctx context.Context, args []string) ([]oapi.SimpleExtension, error) {
	return nil, nil
}

func (s StubbedRegistryClient) PublishExtension(ctx context.Context, ext oapi.PublishExtension) error {
	return nil
}

func (s StubbedRegistryClient) GetVersion(ctx context.Context, name, version string) (*oapi.Extension, error) {
	if name == s.ExtGetVersion.Name {
		for _, pkg := range s.ExtGetVersion.Packages {
			if pkg.Version == version {
				return s.ExtGetVersion, nil
			}
		}

		return nil, registry.ErrExtensionNotFound
	}

	return nil, registry.ErrExtensionNotFound

}

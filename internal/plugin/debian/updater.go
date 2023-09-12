package debian

import (
	"context"
	"fmt"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/log"
)

const (
	gpgkeyURL  = "https://pgxman.github.io/buildkit/pgxman.gpg"
	sourcesURL = "https://pgxman-buildkit-debian.s3.amazonaws.com"
)

type DebianUpdater struct {
	Logger *log.Logger
}

func (u *DebianUpdater) Update(ctx context.Context) error {
	u.Logger.Debug("Downloading buildkit source")
	if err := buildkit.DownloadSource(ctx); err != nil {
		return err
	}

	bt, err := pgxman.DetectExtensionBuilder()
	if err != nil {
		return fmt.Errorf("detect platform: %s", err)
	}

	var (
		prefix   string
		codename string
	)

	switch bt {
	case pgxman.ExtensionBuilderDebianBookworm:
		prefix = "debian"
		codename = "bookworm"
	case pgxman.ExtensionBuilderUbuntuJammy:
		prefix = "ubuntu"
		codename = "jammy"
	default:
		return fmt.Errorf("unsupported platform")
	}

	u.Logger.Debug("Adding apt repositories")
	if err := addAptRepos(
		ctx,
		[]pgxman.AptRepository{
			{
				ID:         "pgxman",
				Types:      []pgxman.AptRepositoryType{pgxman.AptRepositoryTypeDeb},
				URIs:       []string{fmt.Sprintf("%s/%s", sourcesURL, prefix)},
				Suites:     []string{codename},
				Components: []string{"main"},
				SignedKey: pgxman.AptRepositorySignedKey{
					URL:    gpgkeyURL,
					Format: pgxman.AptRepositorySignedKeyFormatGpg,
				},
			},
		},
		u.Logger,
	); err != nil {
		return fmt.Errorf("add apt repos: %w", err)
	}

	return nil
}

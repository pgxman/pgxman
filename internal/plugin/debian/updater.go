package debian

import (
	"context"

	"github.com/pgxman/pgxman"
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
	return addAptRepos(ctx, []pgxman.AptRepository{
		{
			ID:    "pgxman",
			Types: []string{"deb"},
			URIs:  []string{sourcesURL},
			Suites: []pgxman.AptRepositorySuite{
				{
					Suite: "stable",
				},
			},
			Components: []string{"main"},
			SignedKey:  gpgkeyURL,
		},
	})
}

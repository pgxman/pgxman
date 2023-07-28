package debian

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
	"golang.org/x/exp/slog"
)

const (
	gpgkeyURL  = "https://pgxman.github.io/buildkit/pgxman.gpg"
	sourcesURL = "https://pgxman-buildkit-debian.s3.amazonaws.com"
)

type DebianUpdater struct {
	Logger *log.Logger
}

func (u *DebianUpdater) Update(ctx context.Context) error {
	u.Logger.Debug("Downloading buildkit source", slog.String("dir", BuildkitDir))
	if err := downloadBuildkitSource(ctx); err != nil {
		return errBuildkitSource{Err: err}
	}

	u.Logger.Debug("Adding apt repositories")
	if err := addAptRepos(
		ctx,
		[]pgxman.AptRepository{
			{
				ID:         "pgxman",
				Types:      []pgxman.AptRepositoryType{"deb"},
				URIs:       []string{sourcesURL},
				Suites:     []string{"stable"},
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

func downloadBuildkitSource(ctx context.Context) error {
	if err := os.MkdirAll(ConfigDir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(BuildkitDir); err == nil {
		gitFetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
		gitFetchCmd.Dir = BuildkitDir

		if out, err := gitFetchCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git fetch: %w\n%s", err, out)
		}

		gitResetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", "origin/main")
		gitResetCmd.Dir = BuildkitDir

		if out, err := gitResetCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git reset: %w\n%s", err, out)
		}

		return nil
	} else {
		gitCloneCmd := exec.CommandContext(ctx, "git", "clone", "--single-branch", "https://github.com/pgxman/buildkit.git")
		gitCloneCmd.Dir = ConfigDir

		if out, err := gitCloneCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone: %w\n%s", err, out)
		}
	}

	return nil
}

package darwin

import (
	"context"

	"github.com/pgxman/pgxman/internal/buildkit"
	"github.com/pgxman/pgxman/internal/log"
)

type DarwinUpdater struct {
	Logger *log.Logger
}

func (u *DarwinUpdater) Update(ctx context.Context) error {
	u.Logger.Debug("Downloading buildkit source")
	if err := buildkit.DownloadSource(ctx); err != nil {
		return err
	}

	return nil
}

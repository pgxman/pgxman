package debian

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/log"
)

type DebianInstaller struct {
	Logger *log.Logger
}

func (i *DebianInstaller) Install(ctx context.Context, ext []pgxman.InstallExtension) error {
	i.Logger.Debug("Installing extensions", "extensions", ext)

	exts := make([]string, len(ext))
	for _, e := range ext {
		if err := e.Validate(); err != nil {
			return err
		}

		if e.Path != "" {
			exts = append(exts, e.Path)
		} else {
			exts = append(exts, fmt.Sprintf("postgresql-%s-pgxman-%s=%s", e.PGVersion, e.Name, e.Version))
		}
	}

	return i.runAptInstall(ctx, exts...)
}

func (i *DebianInstaller) runAptInstall(ctx context.Context, exts ...string) error {
	for _, ext := range exts {
		i.Logger.Debug("Running apt install", "extension", ext)

		cmd := exec.CommandContext(ctx, "apt", "install", "-y", ext)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

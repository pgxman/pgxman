package debian

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/exp/slog"
)

type DebianUpdater struct {
	Logger *slog.Logger
}

func (u *DebianUpdater) Update(ctx context.Context) error {
	if err := u.downloadFile("https://pgxman.github.io/buildkit/pgxman.asc", "/etc/apt/trusted.gpg.d/pgxman.asc"); err != nil {
		return err
	}

	if err := u.writeAptSource("/etc/apt/sources.list.d/pgxman.list"); err != nil {
		return err
	}

	return u.runAptUpdate(ctx)
}

func (u *DebianUpdater) downloadFile(url, path string) error {
	u.Logger.Debug("Downloading GPG key", slog.String("url", url), slog.String("path", path))

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func (u *DebianUpdater) writeAptSource(path string) error {
	u.Logger.Debug("Writing apt source file", slog.String("path", path))

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	// TODO: Distinguish release codename
	return os.WriteFile(path, []byte(fmt.Sprintf("deb [arch=%s] https://pgxman-buildkit-debian.s3.amazonaws.com stable main", runtime.GOARCH)), 0644)
}

func (u *DebianUpdater) runAptUpdate(ctx context.Context) error {
	u.Logger.Debug("Running apt update")

	cmd := exec.CommandContext(ctx, "apt", "update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

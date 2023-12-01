package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/Masterminds/semver/v3"
)

const (
	MinMajorVersion = 24
)

var (
	ErrDaemonNotRunning = errors.New("docker daemon not running")
	ErrClientNotFound   = errors.New("docker client not found")
	ErrMinVersion       = fmt.Errorf("docker minimum version must be %d", MinMajorVersion)
)

type version struct {
	Client struct {
		Version string
	}

	Server struct {
		Version string
	}
}

func CheckInstall(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.Join(ErrClientNotFound, ErrDaemonNotRunning)
		}

		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			return ErrDaemonNotRunning
		}

		return fmt.Errorf("%s %w", out, err)
	}

	var ver version
	if err := json.Unmarshal(out, &ver); err != nil {
		return ErrDaemonNotRunning
	}

	checkVer := func(v string) error {
		ver, err := semver.StrictNewVersion(v)
		if err != nil {
			return ErrMinVersion
		}

		if ver.Major() < MinMajorVersion {
			return ErrMinVersion
		}

		return nil
	}

	if err := checkVer(ver.Client.Version); err != nil {
		return err
	}
	if err := checkVer(ver.Server.Version); err != nil {
		return err
	}

	return nil
}

package docker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	ErrDockerNotRunning = errors.New("docker daemon not running")
	ErrDockerNotFound   = errors.New("docker client not found")
)

func CheckInstall(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return errors.Join(ErrDockerNotFound, ErrDockerNotRunning)
		}

		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			return ErrDockerNotRunning
		}

		return fmt.Errorf("%s %w", out, err)
	}

	outMap := make(map[string]any)
	if err := json.Unmarshal(out, &outMap); err != nil {
		return ErrDockerNotRunning
	}

	if _, ok := outMap["Server"]; !ok {
		return ErrDockerNotRunning
	}

	return nil
}

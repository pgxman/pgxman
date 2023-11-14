package doctor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

var (
	ErrDockerNotRunning = errors.New("Docker daemon not running")
	ErrDockerNotFound   = errors.New("Docker client not found")
)

func Check(ctx context.Context) error {
	return nil
}

func CheckDocker(ctx context.Context) error {
	cmd := exec.CommandContext(ctx, "docker", "version", "--format", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return ErrDockerNotFound
		}

		if strings.Contains(string(out), "Cannot connect to the Docker daemon") {
			return ErrDockerNotRunning
		}

		return fmt.Errorf("docker error: %s %w", out, err)
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

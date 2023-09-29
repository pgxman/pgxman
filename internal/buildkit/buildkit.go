package buildkit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pgxman/pgxman"
	"sigs.k8s.io/yaml"
)

var (
	configDir   string
	buildkitDir string
)

func init() {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err.Error())
	}

	configDir = filepath.Join(userConfigDir, "pgxman")
	buildkitDir = filepath.Join(configDir, "buildkit")
}

func DownloadSource(ctx context.Context) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(buildkitDir); err == nil {
		gitFetchCmd := exec.CommandContext(ctx, "git", "fetch", "origin")
		gitFetchCmd.Dir = buildkitDir

		if out, err := gitFetchCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git fetch: %w\n%s", err, out)
		}

		gitResetCmd := exec.CommandContext(ctx, "git", "reset", "--hard", "origin/main")
		gitResetCmd.Dir = buildkitDir

		if out, err := gitResetCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git reset: %w\n%s", err, out)
		}

		return nil
	} else {
		gitCloneCmd := exec.CommandContext(ctx, "git", "clone", "--single-branch", "https://github.com/pgxman/buildkit.git")
		gitCloneCmd.Dir = configDir

		if out, err := gitCloneCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone: %w\n%s", err, out)
		}
	}

	return nil
}

func Extensions(ctx context.Context) (map[string]pgxman.Extension, error) {
	if err := DownloadSource(ctx); err != nil {
		return nil, err
	}

	matches, err := filepath.Glob(filepath.Join(buildkitDir, "buildkit", "*.yaml"))
	if err != nil {
		return nil, err
	}

	exts := make(map[string]pgxman.Extension)
	for _, m := range matches {
		b, err := os.ReadFile(m)
		if err != nil {
			return nil, err
		}

		var ext pgxman.Extension
		if err := yaml.Unmarshal(b, &ext); err != nil {
			return nil, err
		}

		exts[ext.Name] = ext
	}

	return exts, nil
}

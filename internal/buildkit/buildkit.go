package buildkit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/config"
	"sigs.k8s.io/yaml"
)

var (
	buildkitDir string
	extsOnce    = sync.OnceValues(extensions)
)

func init() {
	buildkitDir = filepath.Join(config.ConfigDir(), "buildkit")

}

func Extensions() (map[string]pgxman.Extension, error) {
	return extsOnce()
}

func extensions() (map[string]pgxman.Extension, error) {
	if err := downloadSource(context.Background()); err != nil {
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

func downloadSource(ctx context.Context) error {
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
		if err := os.MkdirAll(config.ConfigDir(), 0755); err != nil {
			return err
		}

		gitCloneCmd := exec.CommandContext(ctx, "git", "clone", "--single-branch", "https://github.com/pgxman/buildkit.git")
		gitCloneCmd.Dir = config.ConfigDir()

		if out, err := gitCloneCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone: %w\n%s", err, out)
		}
	}

	return nil
}

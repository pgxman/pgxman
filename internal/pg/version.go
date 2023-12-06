package pg

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/pgxman/pgxman"
)

var (
	regexpPGVersion = regexp.MustCompile(`^PostgreSQL (\d+)`)
)

func DetectVersion(ctx context.Context) (pgxman.PGVersion, error) {
	return pgConfigVersion(ctx, "pg_config")
}

func VersionExists(ctx context.Context, ver pgxman.PGVersion) bool {
	path := fmt.Sprintf("/usr/lib/postgresql/%s/bin/pg_config", ver)
	pgVer, err := pgConfigVersion(ctx, path)
	if err != nil {
		return false
	}

	return pgVer == ver
}

func pgConfigVersion(ctx context.Context, path string) (pgxman.PGVersion, error) {
	cmd := exec.CommandContext(ctx, path, "--version")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return pgxman.PGVersionUnknown, err
	}

	matches := regexpPGVersion.FindStringSubmatch(string(b))
	if len(matches) == 0 {
		return pgxman.PGVersionUnknown, fmt.Errorf("failed to parse pg_config --version output: %s", string(b))
	}

	def := pgxman.PGVersion(matches[1])
	if err := def.Validate(); err != nil {
		return pgxman.PGVersionUnknown, fmt.Errorf("invalid PostgreSQL version: %w", err)
	}

	return def, nil
}

package pg

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pgxman/pgxman"
)

var (
	regexpPGVersion = regexp.MustCompile(`^PostgreSQL (\d+).+\((.+)\s(.+)\)$`)

	ErrParsingPGVersion     = fmt.Errorf("failed to parse pg version")
	ErrUnsupportedPGVersion = fmt.Errorf("unsupported pg version")
	ErrUnsupportedPGDistro  = fmt.Errorf("unsupported pg distribution")
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

	return parsePGVersion(strings.TrimSpace(string(b)))
}

func parsePGVersion(s string) (pgxman.PGVersion, error) {
	matches := regexpPGVersion.FindStringSubmatch(s)
	if len(matches) == 0 {
		return pgxman.PGVersionUnknown, ErrParsingPGVersion
	}

	if !strings.Contains(matches[3], "pgdg") {
		return pgxman.PGVersionUnknown, ErrUnsupportedPGDistro
	}

	def := pgxman.PGVersion(matches[1])
	if err := def.Validate(); err != nil {
		return pgxman.PGVersionUnknown, ErrUnsupportedPGVersion
	}

	return def, nil
}

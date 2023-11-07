package pg

import (
	"context"
	"os/exec"
	"regexp"
	"slices"

	"github.com/pgxman/pgxman"
)

var (
	regexpPGVersion = regexp.MustCompile(`^PostgreSQL (\d+)`)
)

func DetectVersion(ctx context.Context) pgxman.PGVersion {
	cmd := exec.CommandContext(ctx, "pg_config", "--version")
	b, err := cmd.CombinedOutput()
	if err != nil {
		return pgxman.DefaultPGVersion
	}

	matches := regexpPGVersion.FindStringSubmatch(string(b))
	if len(matches) == 0 {
		return pgxman.DefaultPGVersion
	}

	def := pgxman.PGVersion(matches[1])
	if !slices.Contains(pgxman.SupportedPGVersions, def) {
		return pgxman.DefaultPGVersion
	}

	return def
}

package e2etest

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/filepathx"
	"github.com/pgxman/pgxman/internal/log"
	tassert "github.com/stretchr/testify/assert"
	"golang.org/x/exp/slog"
)

func TestBuilder(t *testing.T) {
	assert := tassert.New(t)

	ext := pgxman.NewDefaultExtension()
	ext.Name = "pgvector"
	ext.Description = "pgvector is a PostgreSQL extension for vector similarity search."
	ext.Source = "https://github.com/pgvector/pgvector/archive/refs/tags/v0.4.4.tar.gz"
	ext.Version = "0.4.4"
	ext.Arch = pgxman.SupportedArchs
	ext.Formats = pgxman.SupportedFormats
	// faking the build to speed up the test
	ext.Build = `echo $DSTDIR
echo $PG_CONFIG
echo $PGXS
`
	ext.PGVersions = pgxman.SupportedPGVersions
	ext.Maintainers = []pgxman.Maintainer{
		{
			Name:  "Owen Ou",
			Email: "o@owenou.com",
		},
	}
	ext.BuildImage = flagBuildImage

	extdir := t.TempDir()
	builder := pgxman.NewBuilder(
		pgxman.BuilderOptions{
			ExtDir: extdir,
			Debug:  true,
			// Caching for CI.
			// They are ignored when not running in GitHub Actions.
			CacheFrom: []string{"type=gha"},
			CacheTo:   []string{"type=gha,mode=max"},
		},
	)

	err := builder.Build(context.TODO(), ext)
	assert.NoError(err)

	matches, err := filepathx.WalkMatch(extdir, "*.deb")
	assert.NoError(err)
	assert.Len(matches, 6) // 13, 14, 15 for amd64 & arm64

	for _, match := range matches {
		var (
			match   = match
			debFile = filepath.Base(match)
		)

		if !strings.Contains(debFile, runtime.GOARCH) {
			continue
		}

		t.Run(debFile, func(t *testing.T) {
			t.Parallel()

			assert := tassert.New(t)

			logger := log.NewTextLogger()
			logger = logger.With(slog.String("debfile", debFile))
			w := logger.Writer()

			cmd := exec.Command(
				"docker",
				"run",
				"--rm",
				"-v",
				filepath.Join(extdir, "out")+":/out",
				"ubuntu:22.04",
				"bash",
				"--noprofile",
				"--norc",
				"-eo",
				"pipefail",
				"-c",
				fmt.Sprintf(`export DEBIAN_FRONTEND=noninteractive
apt update
apt install gnupg2 postgresql-common -y
# make sure all pg versions are available
sh /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y
apt update
apt install /out/%s -y
`, debFile),
			)
			cmd.Stdout = w
			cmd.Stderr = w

			err := cmd.Run()
			assert.NoError(err)
		})
	}

}

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
	tassert "github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	assert := tassert.New(t)

	ext := pgxman.NewDefaultExtension()
	ext.Name = "pgvector"
	ext.Description = "pgvector is a PostgreSQL extension for vector similarity search."
	ext.Source = "https://github.com/pgvector/pgvector/archive/refs/tags/v0.5.0.tar.gz"
	ext.Version = "0.5.0"
	ext.License = "PostgreSQL"
	ext.BuildDependencies = []string{"libcurl4-openssl-dev", "pgxman/pgsql-http"}
	ext.RunDependencies = []string{"libcurl4-openssl-dev"}
	ext.Builders = &pgxman.ExtensionBuilders{
		DebianBookworm: &pgxman.AptExtensionBuilder{
			ExtensionBuilder: pgxman.ExtensionBuilder{
				BuildDependencies: []string{"libarrow-dev"},
				Image:             flagDebianBookwormImage,
			},
			AptRepositories: []pgxman.AptRepository{
				{
					ID:         "apache-arrow-debian-bookworm",
					Types:      pgxman.SupportedAptRepositoryTypes,
					URIs:       []string{"https://apache.jfrog.io/artifactory/arrow/debian"},
					Components: []string{"main"},
					Suites:     []string{"bookworm"},
					SignedKey: pgxman.AptRepositorySignedKey{
						URL:    "https://downloads.apache.org/arrow/KEYS",
						Format: pgxman.AptRepositorySignedKeyFormatAsc,
					},
				},
			},
		},
		UbuntuJammy: &pgxman.AptExtensionBuilder{
			ExtensionBuilder: pgxman.ExtensionBuilder{
				BuildDependencies: []string{"libarrow-dev"},
				Image:             flagUbuntuJammyImage,
			},
			AptRepositories: []pgxman.AptRepository{
				{
					ID:         "apache-arrow-ubuntu-jammy",
					Types:      pgxman.SupportedAptRepositoryTypes,
					URIs:       []string{"https://apache.jfrog.io/artifactory/arrow/ubuntu"},
					Components: []string{"main"},
					Suites:     []string{"jammy"},
					SignedKey: pgxman.AptRepositorySignedKey{
						URL:    "https://downloads.apache.org/arrow/KEYS",
						Format: pgxman.AptRepositorySignedKeyFormatAsc,
					},
				},
			},
		},
	}
	ext.Arch = []pgxman.Arch{pgxman.Arch(runtime.GOARCH)} // only build for current arch
	ext.Formats = pgxman.SupportedFormats
	ext.Build = pgxman.Build{
		Main: []pgxman.BuildScript{
			{
				Name: "fake build",
				// faking the build to speed up the test
				Run: `echo $DSTDIR
echo $PG_CONFIG
echo $PGXS
`,
			},
		},
	}

	ext.PGVersions = pgxman.SupportedPGVersions
	ext.Maintainers = []pgxman.Maintainer{
		{
			Name:  "Owen Ou",
			Email: "o@owenou.com",
		},
	}

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
	assert.Len(matches, 4*2) // 13, 14, 15, 16 for current arch only for debian:bookworm & ubuntu:jammy

	for _, match := range matches {
		var (
			match   = match
			debFile = filepath.Base(match)
		)

		if !strings.Contains(debFile, runtime.GOARCH) {
			continue
		}

		var (
			image      string
			pathPrefix string
		)
		if strings.Contains(match, "ubuntu") {
			image = "ubuntu:jammy"
			pathPrefix = "ubuntu/jammy"
		} else if strings.Contains(match, "debian") {
			image = "postgres:14-bookworm"
			pathPrefix = "debian/bookworm"
		} else {
			assert.Failf("unexpected debian package os: %s", match)
		}

		name := fmt.Sprintf("%s-%s", image, debFile)
		name = strings.ReplaceAll(name, ":", "-")
		name = strings.ReplaceAll(name, ".", "-")

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert := tassert.New(t)

			EnsureCleanup(t, func() {
				cmd := exec.Command("docker", "rm", "-f", name)
				_ = cmd.Run()
			})

			cmd := exec.Command(
				"docker",
				"run",
				"--rm",
				"--name",
				name,
				"-e",
				"DEBIAN_FRONTEND=noninteractive",
				"-v",
				filepath.Join(extdir, "out")+":/out",
				"-v",
				flagPGXManBin+":/usr/local/bin/pgxman",
				image,
				"bash",
				"--noprofile",
				"--norc",
				"-eo",
				"pipefail",
				"-c",
				fmt.Sprintf(`export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install ca-certificates gnupg2 postgresql-common git -y
# make sure all pg versions are available
sh /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y
cat <<EOS | pgxman install -f -
apiVersion: v1
extensions:
- name: "pg_ivm"
  version: "1.5.1"
- path: "/out/%s"
pgVersions:
- "14"
EOS
`, filepath.Join(pathPrefix, debFile)),
			)

			b, err := cmd.CombinedOutput()
			assert.NoError(err, string(b))
		})
	}
}

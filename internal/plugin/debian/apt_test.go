package debian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_exitingAptSourceURIs(t *testing.T) {
	assert := assert.New(t)

	sourceDir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(sourceDir, "pgxman-core.sources"),
		[]byte(`
Types: deb
URIs: https://pgxman-buildkit-debian.s3.amazonaws.com
Suites: stable
Components: main
Signed-By: /usr/share/keyrings/pgxman.gpg
`),
		0644,
	)
	assert.NoError(err)

	err = os.WriteFile(
		filepath.Join(sourceDir, "pgxman-pgdg.sources"),
		[]byte(`
Types: deb
URIs: https://apt.postgresql.org/pub/repos/apt
Suites: bookworm-pgdg
Components: main
Signed-By: /usr/share/postgresql-common/pgdg/apt.postgresql.org.gpg

Types: deb
URIs: https://apt.postgresql.org/pub/repos/apt1
Suites: bookworm-pgdg
Components: main
Signed-By: /usr/share/postgresql-common/pgdg/apt.postgresql.org.gpg
`),
		0644,
	)
	assert.NoError(err)

	uris, err := exitingAptSourceURIs(sourceDir)
	assert.NoError(err)

	assert.Equal(
		map[string]struct{}{
			"https://apt.postgresql.org/pub/repos/apt":  struct{}{},
			"https://apt.postgresql.org/pub/repos/apt1": struct{}{},
		},
		uris,
	)
}

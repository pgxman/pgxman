package debian

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_exitingAptSourceHosts(t *testing.T) {
	assert := assert.New(t)

	sourceDir := t.TempDir()

	err := os.WriteFile(
		filepath.Join(sourceDir, "pgxman-core.sources"),
		[]byte(`
Types: deb
URIs: https://apt.pgxman.com
Suites: stable
Components: main
Signed-By: /usr/share/keyrings/pgxman.gpg
`),
		0644,
	)
	assert.NoError(err)

	err = os.WriteFile(
		filepath.Join(sourceDir, "pgdg.list"),
		[]byte(`
deb [ signed-by=/usr/local/share/keyrings/postgres.gpg.asc ] http://apt.postgresql.org/pub/repos/apt2/ bookworm-pgdg main 15
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

	uris, err := exitingAptSourceHosts(sourceDir)
	assert.NoError(err)

	assert.Equal(
		map[string]struct{}{
			"/apt.postgresql.org/pub/repos/apt":  {},
			"/apt.postgresql.org/pub/repos/apt1": {},
			"/apt.postgresql.org/pub/repos/apt2": {},
		},
		uris,
	)
}

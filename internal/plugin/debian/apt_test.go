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

func Test_conflictWithPostgresPackage(t *testing.T) {
	out := `Reading package lists... Done
Building dependency tree... Done
Reading state information... Done
The following NEW packages will be installed:
  postgresql-15-pgxman-pg-stat-statements
0 upgraded, 1 newly installed, 0 to remove and 0 not upgraded.
Need to get 49.5 kB of archives.
After this operation, 164 kB of additional disk space will be used.
Get:1 https://apt-mirror.pgxman.com/debian bookworm/main arm64 postgresql-15-pgxman-pg-stat-statements arm64 15.5.0 [49.5 kB]
Fetched 49.5 kB in 1s (60.9 kB/s)
debconf: delaying package configuration, since apt-utils is not installed
Selecting previously unselected package postgresql-15-pgxman-pg-stat-statements.
(Reading database ... 13384 files and directories currently installed.)
Preparing to unpack .../postgresql-15-pgxman-pg-stat-statements_15.5.0_arm64.deb ...
Unpacking postgresql-15-pgxman-pg-stat-statements (15.5.0) ...
dpkg: error processing archive /var/cache/apt/archives/postgresql-15-pgxman-pg-stat-statements_15.5.0_arm64.deb (--unpack):
 trying to overwrite '/usr/lib/postgresql/15/lib/bitcode/pg_stat_statements/pg_stat_statements.bc', which is also in package postgresql-15 15.5-1.pgdg120+1
Errors were encountered while processing:
 /var/cache/apt/archives/postgresql-15-pgxman-pg-stat-statements_15.5.0_arm64.deb
E: Sub-process /usr/bin/dpkg returned an error code (1)
	`

	assert.True(t, conflictDebPkg(out))
}

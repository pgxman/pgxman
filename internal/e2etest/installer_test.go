package e2etest

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebianInstaller_CLI(t *testing.T) {
	exts := []struct {
		Extension string
		Version   string
		PGVersion string
	}{
		{
			Extension: "pgvector",
			Version:   "0.5.0",
			PGVersion: "15",
		},
		{
			Extension: "pgvector",
			Version:   "",
			PGVersion: "15",
		},
		{
			Extension: "pgvector",
			Version:   "latest",
			PGVersion: "15",
		},
	}

	for _, ext := range exts {
		ext := ext

		name := fmt.Sprintf("%s-%s-%s", ext.Extension, ext.Version, ext.PGVersion)
		var installArg string
		if ext.Version == "" {
			installArg = fmt.Sprintf("%s@%s", ext.Extension, ext.PGVersion)
		} else {
			installArg = fmt.Sprintf("%s=%s@%s", ext.Extension, ext.Version, ext.PGVersion)
		}

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert := assert.New(t)
			cmd := exec.Command(
				"docker",
				"run",
				"--rm",
				"--name",
				name,
				"-e",
				"DEBIAN_FRONTEND=noninteractive",
				"-v",
				flagPGXManBin+":/usr/local/bin/pgxman",
				fmt.Sprintf("postgres:%s", ext.PGVersion),
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
pgxman install %s --yes
`, installArg),
			)

			b, err := cmd.CombinedOutput()
			assert.NoError(err, string(b))
		})
	}

}

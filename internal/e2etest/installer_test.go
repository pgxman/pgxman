package e2etest

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/stretchr/testify/assert"
)

func TestDebianInstaller_Install(t *testing.T) {
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
		{
			Extension: "pgvector",
		},
		{
			Extension: "pgvector",
			Version:   "latest",
		},
	}

	for _, ext := range exts {
		ext := ext

		installArg := ext.Extension
		if ext.Version != "" {
			installArg += "=" + ext.Version
		}
		if ext.PGVersion != "" {
			installArg += " --pg " + ext.PGVersion
		}

		name := strings.ReplaceAll(installArg, "=", "_")
		name = strings.ReplaceAll(name, " --pg ", "_")

		pgv := ext.PGVersion
		if pgv == "" {
			pgv = string(pgxman.SupportedLatestPGVersion)
		}
		image := fmt.Sprintf("postgres:%s", pgv)

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
pgxman install %s --yes
`, installArg),
			)

			b, err := cmd.CombinedOutput()
			assert.NoError(err, string(b))
		})
	}

}

func TestDebianInstaller_Upgrade(t *testing.T) {
	assert := assert.New(t)
	cmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"--name",
		"pgxman-upgrade",
		"-e",
		"DEBIAN_FRONTEND=noninteractive",
		"-v",
		flagPGXManBin+":/usr/local/bin/pgxman",
		"postgres:16",
		"bash",
		"--noprofile",
		"--norc",
		"-eo",
		"pipefail",
		"-c",
		`export DEBIAN_FRONTEND=noninteractive
apt-get update
apt-get install ca-certificates gnupg2 postgresql-common git -y
# make sure all pg versions are available
sh /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh -y
pgxman install pgvector=0.5.0 --yes
# upgrade
pgxman upgrade pgvector=0.5.1 --yes
# downgrade
pgxman upgrade pgvector=0.5.0 --yes
`,
	)

	b, err := cmd.CombinedOutput()
	assert.NoError(err, string(b))

}

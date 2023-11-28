package e2etest

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/pgxman/pgxman/internal/container"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func TestContainer(t *testing.T) {
	assert := assert.New(t)

	configDir := t.TempDir()
	c := container.NewContainer(
		container.WithRunnerImage(flagRunnerPostgres15Image),
		container.WithConfigDir(configDir),
	)
	wantExt := pgxman.InstallExtension{
		PackExtension: pgxman.PackExtension{
			Name:    "pgvector",
			Version: "0.5.1",
		},
		PGVersion: pgxman.PGVersion15,
	}

	info, err := c.Install(context.TODO(), wantExt)
	assert.NoError(err)

	b, err := os.ReadFile(filepath.Join(info.RunnerDir, "pgxman.yaml"))
	assert.NoError(err)

	var gotFile pgxman.Pack
	err = yaml.Unmarshal(b, &gotFile)
	assert.NoError(err)
	assert.Equal(pgxman.Pack{
		APIVersion: pgxman.DefaultPackAPIVersion,
		Extensions: []pgxman.PackExtension{
			{
				Name:      wantExt.Name,
				Version:   wantExt.Version,
				Overwrite: true,
			},
		},
		Postgres: info.Postgres,
	}, gotFile)

	err = c.Teardown(context.TODO(), pgxman.PGVersion15)
	assert.NoError(err)
}

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

	c := container.NewContainer()
	wantFile := &pgxman.PGXManfile{
		APIVersion: pgxman.DefaultPGXManfileAPIVersion,
		PGVersions: []pgxman.PGVersion{pgxman.PGVersion15},
		Extensions: []pgxman.InstallExtension{
			{
				Name:    "pgvector",
				Version: "0.5.1",
			},
		},
	}

	info, err := c.Install(context.TODO(), wantFile)
	assert.NoError(err)

	b, err := os.ReadFile(filepath.Join(info.RunnerDir, "pgxman.yaml"))
	assert.NoError(err)

	var gotFile pgxman.PGXManfile
	err = yaml.Unmarshal(b, &gotFile)
	assert.NoError(err)
	assert.Equal(wantFile, &gotFile)

	err = c.Teardown(context.TODO(), pgxman.PGVersion15)
	assert.NoError(err)
}

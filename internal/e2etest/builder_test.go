package e2etest

import (
	"context"
	"testing"

	"github.com/hydradatabase/pgxman"
	"github.com/hydradatabase/pgxman/internal/filepathx"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	assert := assert.New(t)

	ext := pgxman.NewDefaultExtension()
	ext.Name = "pgvector"
	ext.Description = "pgvector is a PostgreSQL extension for vector similarity search."
	ext.Source = "https://github.com/pgvector/pgvector/archive/refs/tags/v0.4.2.tar.gz"
	ext.Version = "0.4.2"
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
	builder := pgxman.NewBuilder(extdir, true)

	err := builder.Build(context.TODO(), ext)
	assert.NoError(err)

	matches, err := filepathx.WalkMatch(extdir, "*.deb")
	assert.NoError(err)
	assert.Len(matches, 6) // 13, 14, 15 for amd64 & arm64
}

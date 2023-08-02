package pgxman

import (
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/stretchr/testify/assert"
)

func Test_parseInstallExtension(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Name   string
		Arg    string
		GotExt *pgxman.PGXManFile
		Err    error
	}{
		{
			Name: "valid with one pgversion",
			Arg:  "pgvector=0.4.4@14",
			GotExt: &pgxman.PGXManFile{
				APIVersion: pgxman.DefaultInstallExtensionsAPIVersion,
				Extensions: []pgxman.InstallExtension{
					{
						Name:    "pgvector",
						Version: "0.4.4",
					},
				},
				PGVersions: []pgxman.PGVersion{pgxman.PGVersion14},
			},
		},
		{
			Name: "valid with sha as version",
			Arg:  "parquet_s3_fdw=5298b7f0254923f52d15e554ec8a5fdc0474f059@14",
			GotExt: &pgxman.PGXManFile{
				APIVersion: pgxman.DefaultInstallExtensionsAPIVersion,
				Extensions: []pgxman.InstallExtension{
					{
						Name:    "parquet_s3_fdw",
						Version: "5298b7f0254923f52d15e554ec8a5fdc0474f059",
					},
				},
				PGVersions: []pgxman.PGVersion{pgxman.PGVersion14},
			},
		},
		{
			Name: "valid with multiple pgversions",
			Arg:  "pgvector=0.4.4@14,15",
			GotExt: &pgxman.PGXManFile{
				APIVersion: pgxman.DefaultInstallExtensionsAPIVersion,
				Extensions: []pgxman.InstallExtension{
					{
						Name:    "pgvector",
						Version: "0.4.4",
					},
				},
				PGVersions: []pgxman.PGVersion{pgxman.PGVersion14, pgxman.PGVersion15},
			},
		},
		{
			Name: "invalid",
			Arg:  "pgvector=0.4.4@",
			Err:  errInvalidExtensionFormat{Arg: "pgvector=0.4.4@"},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			gotExt, err := parseInstallExtensions(c.Arg)
			assert.Equal(c.Err, err)
			assert.Equal(c.GotExt, gotExt)
		})
	}

}

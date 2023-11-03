package pgxman

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/stretchr/testify/assert"
)

func Test_parseInstallExtension(t *testing.T) {
	assert := assert.New(t)

	debFile := filepath.Join(t.TempDir(), "extension.deb")
	err := os.WriteFile(debFile, []byte{}, 0644)
	assert.NoError(err)

	cases := []struct {
		Name   string
		Arg    string
		GotExt *pgxman.InstallExtension
		Err    error
	}{
		{
			Name: "valid with name & version",
			Arg:  "pgvector=0.5.0",
			GotExt: &pgxman.InstallExtension{
				Name:    "pgvector",
				Version: "0.5.0",
			},
		},
		{
			Name: "valid with sha as version",
			Arg:  "parquet_s3_fdw=5298b7f0254923f52d15e554ec8a5fdc0474f059",
			GotExt: &pgxman.InstallExtension{
				Name:    "parquet_s3_fdw",
				Version: "5298b7f0254923f52d15e554ec8a5fdc0474f059",
			},
		},
		{
			Name: "valid with empty version",
			Arg:  "pgvector=",
			GotExt: &pgxman.InstallExtension{
				Name:    "pgvector",
				Version: "",
			},
		},
		{
			Name: "valid with latest as version",
			Arg:  "pgvector=latest",
			GotExt: &pgxman.InstallExtension{
				Name:    "pgvector",
				Version: "latest",
			},
		},
		{
			Name: "valid with only name",
			Arg:  "pgvector",
			GotExt: &pgxman.InstallExtension{
				Name:    "pgvector",
				Version: "",
			},
		},
		{
			Name: "valid file path",
			Arg:  debFile,
			GotExt: &pgxman.InstallExtension{
				Path: debFile,
			},
		},
		{
			Name: "invalid",
			Arg:  "pgvector=0.5.0@",
			Err:  errInvalidExtensionFormat{Arg: "pgvector=0.5.0@"},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			gotExt, err := parseInstallExtension(c.Arg)
			assert.Equal(c.Err, err)
			assert.Equal(c.GotExt, gotExt)
		})
	}

}

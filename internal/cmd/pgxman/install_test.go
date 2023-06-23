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
		GotExt []pgxman.InstallExtension
		Err    error
	}{
		{
			Name: "valid with one pgversion",
			Arg:  "pgvector=0.4.4@14",
			GotExt: []pgxman.InstallExtension{
				{
					Name:      "pgvector",
					Version:   "0.4.4",
					PGVersion: pgxman.PGVersion14,
				},
			},
		},
		{
			Name: "valid with multiple pgversions",
			Arg:  "pgvector=0.4.4@14,15",
			GotExt: []pgxman.InstallExtension{
				{
					Name:      "pgvector",
					Version:   "0.4.4",
					PGVersion: pgxman.PGVersion14,
				},
				{
					Name:      "pgvector",
					Version:   "0.4.4",
					PGVersion: pgxman.PGVersion15,
				},
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

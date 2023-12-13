package pg

import (
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/stretchr/testify/assert"
)

func Test_parsePGVersion(t *testing.T) {
	cases := []struct {
		Name      string
		Str       string
		WantPGVer pgxman.PGVersion
		WantErr   error
	}{
		{
			Name:      "happy path",
			Str:       "PostgreSQL 16.1 (Debian 16.1-1.pgdg120+1)",
			WantPGVer: pgxman.PGVersion16,
		},
		{
			Name:      "unsupported pg distro",
			Str:       "PostgreSQL 14.10 (Ubuntu 14.10-0ubuntu0.22.04.1)",
			WantPGVer: pgxman.PGVersionUnknown,
			WantErr:   ErrUnsupportedPGDistro,
		},
		{
			Name:      "unsupported pg version",
			Str:       "PostgreSQL 10.1 (Debian 16.1-1.pgdg120+1)",
			WantPGVer: pgxman.PGVersionUnknown,
			WantErr:   ErrUnsupportedPGVersion,
		},
		{
			Name:      "malformed pg version",
			Str:       "PostgreSQL 1234",
			WantPGVer: pgxman.PGVersionUnknown,
			WantErr:   ErrParsingPGVersion,
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			assert := assert.New(t)

			ver, err := parsePGVersion(c.Str)
			assert.Equal(c.WantPGVer, ver)
			assert.Equal(c.WantErr, err)
		})
	}
}

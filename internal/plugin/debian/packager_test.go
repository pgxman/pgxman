package debian

import (
	"bytes"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/stretchr/testify/assert"
)

func Test_debianPackageTemplater(t *testing.T) {
	assert := assert.New(t)

	ext := pgxman.Extension{
		Name:        "pgvector",
		Maintainers: []pgxman.Maintainer{{Name: "Owen Ou", Email: "o@hydra.so"}},
		PGVersions:  []pgxman.PGVersion{pgxman.PGVersion14},
		ExtensionOverridable: pgxman.ExtensionOverridable{
			BuildDependencies: []string{"libxml2", "pgxman/multicorn"},
			RunDependencies:   []string{"libxml2", "pgxman/multicorn"},
		},
	}

	cases := []struct {
		Name        string
		Content     string
		WantContent string
	}{
		{
			Name:        "fields",
			Content:     `{{ .Name }}`,
			WantContent: `pgvector`,
		},
		{
			Name:        "build deps",
			Content:     `{{ .BuildDeps }}`,
			WantContent: "debhelper (>= 9), postgresql-server-dev-all (>= 158~), libxml2, postgresql-PGVERSION-pgxman-multicorn",
		},
		{
			Name:        "deps",
			Content:     `{{ .Deps }}`,
			WantContent: "${shlibs:Depends}, ${misc:Depends}, libxml2, postgresql-PGVERSION-pgxman-multicorn",
		},
		{
			Name:        "maintainers",
			Content:     `{{ .Maintainers }}`,
			WantContent: "Owen Ou <o@hydra.so>",
		},
		{
			Name:        "pg versions",
			Content:     `{{ .PGVersions }}`,
			WantContent: "[14]",
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			buf := bytes.NewBuffer(nil)

			err := debianPackageTemplater{ext}.Render([]byte(c.Content), buf)
			assert.NoError(err)
			assert.Equal(c.WantContent, buf.String())
		})
	}
}

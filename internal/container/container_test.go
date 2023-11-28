package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func Test_mergePackFile(t *testing.T) {
	cases := []struct {
		Name             string
		ExistingPackFile *pgxman.Pack
		NewPackFile      *pgxman.Pack
		WantPackFile     *pgxman.Pack
	}{
		{
			Name: "no existing pack file",
			NewPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
			WantPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
		},
		{
			Name: "merge different extensions",
			ExistingPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
			NewPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pg_ivm",
						Version: "1.0.0",
					},
				},
			},
			WantPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pg_ivm",
						Version: "1.0.0",
					},
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
		},
		{
			Name: "override existing extensions",
			ExistingPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
			NewPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "1.0.1",
					},
				},
			},
			WantPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "1.0.1",
					},
				},
			},
		},
		{
			Name: "merge extension with paths",
			ExistingPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name: "pgvector",
						Path: "path1",
					},
				},
			},
			NewPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "path2",
					},
				},
			},
			WantPackFile: &pgxman.Pack{
				Extensions: []pgxman.PackExtension{
					{
						Name:    "pgvector",
						Version: "path2",
					},
				},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			assert := assert.New(t)

			dir := t.TempDir()
			if c.ExistingPackFile != nil {
				b, err := yaml.Marshal(c.ExistingPackFile)
				assert.NoError(err)

				err = os.WriteFile(filepath.Join(dir, "pgxman.yaml"), b, 0644)
				assert.NoError(err)
			}

			err := mergePackFile(c.NewPackFile, dir)
			assert.NoError(err)

			b, err := os.ReadFile(filepath.Join(dir, "pgxman.yaml"))
			assert.NoError(err)

			var got pgxman.Pack
			err = yaml.Unmarshal(b, &got)
			assert.NoError(err)
			assert.Equal(c.WantPackFile, &got)
		})
	}
}

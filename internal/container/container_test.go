package container

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pgxman/pgxman"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"
)

func Test_mergeBundleFile(t *testing.T) {
	cases := []struct {
		Name               string
		ExistingBundleFile *pgxman.Bundle
		NewBundleFile      *pgxman.Bundle
		WantBundleFile     *pgxman.Bundle
	}{
		{
			Name: "no existing bundle file",
			NewBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
			WantBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
		},
		{
			Name: "merge different extensions",
			ExistingBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
			NewBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pg_ivm",
						Version: "1.0.0",
					},
				},
			},
			WantBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
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
			ExistingBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pgvector",
						Version: "1.0.0",
					},
				},
			},
			NewBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pgvector",
						Version: "1.0.1",
					},
				},
			},
			WantBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pgvector",
						Version: "1.0.1",
					},
				},
			},
		},
		{
			Name: "merge extension with paths",
			ExistingBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name: "pgvector",
						Path: "path1",
					},
				},
			},
			NewBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
					{
						Name:    "pgvector",
						Version: "path2",
					},
				},
			},
			WantBundleFile: &pgxman.Bundle{
				Extensions: []pgxman.BundleExtension{
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
			if c.ExistingBundleFile != nil {
				b, err := yaml.Marshal(c.ExistingBundleFile)
				assert.NoError(err)

				err = os.WriteFile(filepath.Join(dir, "pgxman.yaml"), b, 0644)
				assert.NoError(err)
			}

			err := mergeBundleFile(c.NewBundleFile, dir)
			assert.NoError(err)

			b, err := os.ReadFile(filepath.Join(dir, "pgxman.yaml"))
			assert.NoError(err)

			var got pgxman.Bundle
			err = yaml.Unmarshal(b, &got)
			assert.NoError(err)
			assert.Equal(c.WantBundleFile, &got)
		})
	}
}

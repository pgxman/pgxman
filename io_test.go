package pgxman

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_overrideExtension(t *testing.T) {
	cases := []struct {
		Name      string
		Ext       Extension
		Overrides map[string]any
		WantExt   Extension
	}{
		{
			Name: "no overrides",
			Ext: Extension{
				ExtensionCommon: ExtensionCommon{
					Name: "pgvector",
					Maintainers: []Maintainer{
						{
							Name: "foo",
						},
					},
				},
				PGVersions: []PGVersion{"13", "14", "15"},
				Overrides: &ExtensionOverrides{
					PGVersions: map[PGVersion]ExtensionOverridable{
						PGVersion13: {
							Source: "source13",
						},
						PGVersion14: {
							Source: "source14",
						},
						PGVersion15: {
							Source: "source15",
						},
					},
				},
			},
			Overrides: nil,
			WantExt: Extension{
				ExtensionCommon: ExtensionCommon{
					Name: "pgvector",
					Maintainers: []Maintainer{
						{
							Name: "foo",
						},
					},
				},
				PGVersions: []PGVersion{"13", "14", "15"},
				Overrides: &ExtensionOverrides{
					PGVersions: map[PGVersion]ExtensionOverridable{
						PGVersion13: {
							Source: "source13",
						},
						PGVersion14: {
							Source: "source14",
						},
						PGVersion15: {
							Source: "source15",
						},
					},
				},
			},
		},
		{
			Name: "override pg version",
			Ext: Extension{
				ExtensionCommon: ExtensionCommon{
					Name: "pgvector",
					Maintainers: []Maintainer{
						{
							Name: "foo",
						},
					},
				},
				PGVersions: []PGVersion{"13", "14", "15"},
				Overrides: &ExtensionOverrides{
					PGVersions: map[PGVersion]ExtensionOverridable{
						PGVersion13: {
							Source: "source13",
						},
						PGVersion14: {
							Source: "source14",
						},
						PGVersion15: {
							Source: "source15",
						},
					},
				},
			},
			Overrides: map[string]any{
				"maintainers": []map[string]any{
					{
						"name": "Owen",
					},
				},
				"pgVersions": []string{"13"},
			},
			WantExt: Extension{
				ExtensionCommon: ExtensionCommon{
					Name: "pgvector",
					Maintainers: []Maintainer{
						{
							Name: "Owen",
						},
					},
				},
				PGVersions: []PGVersion{"13"},
				Overrides: &ExtensionOverrides{
					PGVersions: map[PGVersion]ExtensionOverridable{
						PGVersion13: {
							Source: "source13",
						},
					},
				},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			assert := assert.New(t)

			gotExt, err := overrideExtension(c.Ext, c.Overrides)
			assert.NoError(err)
			assert.Equal(c.WantExt, gotExt)
		})
	}
}

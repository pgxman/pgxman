package pgxman

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtension_ParseSource(t *testing.T) {
	assert := assert.New(t)

	tests := []struct {
		name    string
		ext     Extension
		wantExt ExtensionSource
		wantErr bool
	}{
		{
			name: "http source",
			ext: Extension{
				ExtensionOverridable: ExtensionOverridable{
					Source: "http://example.com/test.tar.gz",
				},
			},
			wantExt: &httpExtensionSource{URL: "http://example.com/test.tar.gz"},
		},
		{
			name: "file source",
			ext: Extension{
				ExtensionOverridable: ExtensionOverridable{
					Source: "file:///tmp/test.tar.gz",
				},
			},
			wantExt: &fileExtensionSource{Dir: "/tmp/test.tar.gz"},
		},
		{
			name: "invalid source",
			ext: Extension{
				ExtensionOverridable: ExtensionOverridable{
					Source: "ftp://example.com/test.tar.gz",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotExt, gotErr := tt.ext.ParseSource()

			assert.Equal(tt.wantErr, gotErr != nil)
			assert.Equal(tt.wantExt, gotExt)
		})
	}
}

func TestExtension_Packages(t *testing.T) {
	cases := []struct {
		Name         string
		Ext          Extension
		WantPackages []ExtensionPackage
	}{
		{
			Name: "no overrides",
			Ext: Extension{
				ExtensionCommon: ExtensionCommon{
					Name: "pg_stat_statements",
				},
				ExtensionOverridable: ExtensionOverridable{
					Source: "source",
				},
				PGVersions: []PGVersion{PGVersion15, PGVersion16},
			},
			WantPackages: []ExtensionPackage{
				{
					ExtensionCommon: ExtensionCommon{
						Name: "pg_stat_statements",
					},
					ExtensionOverridable: ExtensionOverridable{
						Source: "source",
					},
					PGVersion: PGVersion15,
				},
				{
					ExtensionCommon: ExtensionCommon{
						Name: "pg_stat_statements",
					},
					ExtensionOverridable: ExtensionOverridable{
						Source: "source",
					},
					PGVersion: PGVersion16,
				},
			},
		},
		{
			Name: "have overrides",
			Ext: Extension{
				ExtensionCommon: ExtensionCommon{
					Name: "pg_stat_statements",
				},
				ExtensionOverridable: ExtensionOverridable{
					Source: "source",
				},
				PGVersions: []PGVersion{PGVersion15, PGVersion16},
				Overrides: &ExtensionOverrides{
					PGVersions: map[PGVersion]ExtensionOverridable{
						PGVersion16: {
							Source: "source16",
						},
						PGVersion15: {
							Version: "15.5.0",
						},
					},
				},
			},
			WantPackages: []ExtensionPackage{
				{
					ExtensionCommon: ExtensionCommon{
						Name: "pg_stat_statements",
					},
					ExtensionOverridable: ExtensionOverridable{
						Source:  "source",
						Version: "15.5.0",
					},
					PGVersion: PGVersion15,
				},
				{
					ExtensionCommon: ExtensionCommon{
						Name: "pg_stat_statements",
					},
					ExtensionOverridable: ExtensionOverridable{
						Source: "source16",
					},
					PGVersion: PGVersion16,
				},
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			assert := assert.New(t)
			assert.Equal(c.WantPackages, c.Ext.Packages())
		})
	}
}

func TestExtension_Validate(t *testing.T) {
	assert := assert.New(t)

	ext := Extension{
		PGVersions: []PGVersion{PGVersion16},
	}
	err := ext.Validate()
	assert.Error(err)
	assert.NotContains(err.Error(), "PostgreSQL 16 config has errors")
	assert.Contains(err.Error(), "version is required")

	ext = Extension{
		PGVersions: []PGVersion{PGVersion16},
		Overrides: &ExtensionOverrides{
			PGVersions: map[PGVersion]ExtensionOverridable{
				PGVersion16: {
					Version: "1.2.3",
				},
			},
		},
	}
	err = ext.Validate()
	assert.Error(err)
	assert.Contains(err.Error(), "PostgreSQL 16 config has errors")
	assert.NotContains(err.Error(), "version is required")

	ext = Extension{
		PGVersions: []PGVersion{PGVersion15},
		Overrides: &ExtensionOverrides{
			PGVersions: map[PGVersion]ExtensionOverridable{
				PGVersion16: {
					Version: "1.2.3",
				},
			},
		},
	}
	err = ext.Validate()
	assert.Error(err)
	assert.Contains(err.Error(), "overriding PostgreSQL 16 config but \"16\" is not in `pgVersions`")
}

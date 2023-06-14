package pgxman

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_overrideYamlFields(t *testing.T) {
	assert := assert.New(t)

	b, err := overrideYamlFields([]byte(`apiVersion: v1
buildImage: foo
maintainers:
- name: foo
pgVersions:
- "10"
- "11"
- "12"
`), map[string]any{
		"buildImage": "bar",
		"maintainers": []map[string]any{
			{
				"name": "Owen",
			},
		},
		"pgVersions": []string{"11", "12"},
	})
	assert.NoError(err)
	assert.Equal(`apiVersion: v1
buildImage: bar
maintainers:
- name: Owen
pgVersions:
- "11"
- "12"
`, string(b))
}

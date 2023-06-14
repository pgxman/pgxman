package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_ParseMapFlag(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Name string
		In   map[string]string
		Out  map[string]any
	}{
		{
			Name: "string value",
			In: map[string]string{
				"foo": "bar",
			},
			Out: map[string]any{
				"foo": "bar",
			},
		},
		{
			Name: "slice value",
			In: map[string]string{
				"foo": "[bar,baz]",
			},
			Out: map[string]any{
				"foo": []string{"bar", "baz"},
			},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			out := ParseMapFlag(c.In)
			assert.Equal(c.Out, out)
		})
	}

}

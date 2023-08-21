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
		{
			Name: "nested keys",
			In: map[string]string{
				"a.b": "[bar,baz]",
			},
			Out: map[string]any{
				"a": map[string]any{"b": []string{"bar", "baz"}},
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

func Test_setRecursiveMap(t *testing.T) {
	assert := assert.New(t)

	cases := []struct {
		Name string
		Keys []string
		Want map[string]any
	}{
		{
			Name: "no key",
			Keys: []string{},
			Want: map[string]any{},
		},
		{
			Name: "single key",
			Keys: []string{"a"},
			Want: map[string]any{"a": "value"},
		},
		{
			Name: "nested keys",
			Keys: []string{"a", "b", "c"},
			Want: map[string]any{"a": map[string]any{"b": map[string]any{"c": "value"}}},
		},
	}

	for _, c := range cases {
		c := c

		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			m := make(map[string]any)
			setNestedMap(m, c.Keys, "value")

			assert.Equal(c.Want, m)
		})
	}
}

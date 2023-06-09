package template

import "io"

type Template interface {
	Apply(content []byte, out io.Writer) error
}

package pgxman

import (
	"context"
)

type PackagerOptions struct {
	WorkDir string
	Debug   bool
}

type Packager interface {
	Package(ctx context.Context, ext Extension, opts PackagerOptions) error
}

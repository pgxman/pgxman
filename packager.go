package pgxman

import (
	"context"
)

type PackagerOptions struct {
	WorkDir  string
	Parallel int
	Debug    bool
}

type Packager interface {
	Init(ctx context.Context, ext Extension, opts PackagerOptions) error
	Pre(ctx context.Context, ext Extension, opts PackagerOptions) error
	Main(ctx context.Context, ext Extension, opts PackagerOptions) error
	Post(ctx context.Context, ext Extension, opts PackagerOptions) error
}

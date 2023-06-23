package pgxman

import (
	"context"
)

type Updater interface {
	Update(ctx context.Context) error
}

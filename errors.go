package pgxman

import "fmt"

var (
	ErrRootAccessRequired = fmt.Errorf("root access is required")
	ErrConflictExtension  = fmt.Errorf("conflict extension")
)

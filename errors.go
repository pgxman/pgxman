package pgxman

import "fmt"

var (
	ErrRootAccessRequired = fmt.Errorf("root access is required")
)

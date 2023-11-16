package pgxman

import "fmt"

var (
	Version = "dev"
)

func ImageTag() string {
	if Version == "dev" {
		return "main"
	}

	return fmt.Sprintf("v%s", Version)
}

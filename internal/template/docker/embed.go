package docker

import (
	"embed"
)

//go:embed all:*
var FS embed.FS

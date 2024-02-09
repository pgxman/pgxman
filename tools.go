//go:build tools

package main

//go:generate go install github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen

import (
	_ "github.com/deepmap/oapi-codegen/v2/cmd/oapi-codegen"
)

package config

import (
	"os"
	"path/filepath"
)

func ConfigDir() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		panic(err.Error())
	}

	return filepath.Join(userConfigDir, "pgxman")
}
